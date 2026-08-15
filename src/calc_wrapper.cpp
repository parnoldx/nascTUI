#include <string>
#include <libqalculate/Calculator.h>
#include <libqalculate/MathStructure.h>
#include <libqalculate/Function.h>
#include <libqalculate/Variable.h>
#include <stdlib.h>
#include <string.h>
#include <locale.h>
#include <mutex>
#include <algorithm>
#include <cctype>

using namespace std;

static bool calculator_initialized = false;
static std::mutex calculator_mutex;

static char* strdup_safe(const std::string& s) {
    char* c = (char*)malloc(s.length() + 1);
    if (!c) {
        return nullptr;
    }
    memcpy(c, s.c_str(), s.length() + 1);
    return c;
}

static std::string toLowerCopy(std::string s) {
    std::transform(s.begin(), s.end(), s.begin(), [](unsigned char c) {
        return static_cast<char>(std::tolower(c));
    });
    return s;
}

static void rtrim(std::string& s) {
    while (!s.empty() && std::isspace(static_cast<unsigned char>(s.back()))) {
        s.pop_back();
    }
}

static bool hasEnding(const std::string& fullString, const std::string& ending) {
    std::string full = toLowerCopy(fullString);
    rtrim(full);
    std::string end = toLowerCopy(ending);
    if (full.length() >= end.length()) {
        return full.compare(full.length() - end.length(), end.length(), end) == 0;
    }
    return false;
}

static void load_currencies() {
    if (calculator) {
        calculator->loadExchangeRates();
    }
}

static void initialize_calculator() {
    std::lock_guard<std::mutex> lock(calculator_mutex);
    if (calculator_initialized) return;

    setlocale(LC_ALL, "");

    if (!calculator) {
        calculator = new Calculator();
    }

    // Treat comma as a decimal separator (European-style input).
    // unlocalizeExpression still honors the process locale for output.
    calculator->useDecimalComma();

    calculator->loadGlobalDefinitions();
    calculator->loadLocalDefinitions();
    load_currencies();

    calculator->useIntervalArithmetic(false);

    calculator_initialized = true;
}

static PrintOptions getPrintOptions(const std::string& input) {
    PrintOptions printops;
    printops.multiplication_sign = MULTIPLICATION_SIGN_ASTERISK;
    printops.number_fraction_format = FRACTION_DECIMAL;
    printops.max_decimals = 9;
    printops.use_max_decimals = true;
    printops.use_unicode_signs = true;
    printops.use_unit_prefixes = false;
    struct lconv * lc  = localeconv();
    printops.decimalpoint_sign = lc->decimal_point;

    if (hasEnding(input, "to hex")) {
        printops.base = BASE_HEXADECIMAL;
    } else if (hasEnding(input, "to bin")) {
        printops.base = BASE_BINARY;
    } else if (hasEnding(input, "to dec")) {
        printops.base = BASE_DECIMAL;
    } else if (hasEnding(input, "to oct")) {
        printops.base = BASE_OCTAL;
    } else if (hasEnding(input, "to duo")) {
        printops.base = BASE_DUODECIMAL;
    } else if (hasEnding(input, "to roman")) {
        printops.base = BASE_ROMAN_NUMERALS;
    } else if (hasEnding(input, "to bijective")) {
        printops.base = BASE_BIJECTIVE_26;
    } else if (hasEnding(input, "to sexa")) {
        printops.base = BASE_SEXAGESIMAL;
    } else if (hasEnding(input, "to fp32")) {
        printops.base = BASE_FP32;
    } else if (hasEnding(input, "to fp64")) {
        printops.base = BASE_FP64;
    } else if (hasEnding(input, "to fp16")) {
        printops.base = BASE_FP16;
    } else if (hasEnding(input, "to fp80")) {
        printops.base = BASE_FP80;
    } else if (hasEnding(input, "to fp128")) {
        printops.base = BASE_FP128;
    } else if (hasEnding(input, "to time")) {
        printops.base = BASE_TIME;
    } else if (hasEnding(input, "to unicode")) {
        printops.base = BASE_UNICODE;
    } else if (hasEnding(input, "to utc") || hasEnding(input, "to gmt")) {
        printops.time_zone = TIME_ZONE_UTC;
    } else if (hasEnding(input, "to cet")) {
        printops.time_zone = TIME_ZONE_CUSTOM;
        printops.custom_time_zone = 60;
    }

    return printops;
}

extern "C" {
    bool update_exchange_rates_if_needed() {
        initialize_calculator();

        {
            std::lock_guard<std::mutex> lock(calculator_mutex);
            if (!calculator_initialized || !calculator) {
                return false;
            }
            // Check the default source regardless of whether a currency
            // conversion has already been evaluated in this session.
            if (!calculator->checkExchangeRatesDate(7, false, false, -1)) {
                return false;
            }
        }

        // Network I/O must not hold calculator_mutex or the UI freezes.
        bool success = calculator->fetchExchangeRates(15, -1);
        if (success) {
            std::lock_guard<std::mutex> lock(calculator_mutex);
            calculator->loadExchangeRates();
        }
        return success;
    }

    char* calculate_expression(const char* expression) {
        initialize_calculator();

        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) {
            return strdup_safe("Error");
        }

        EvaluationOptions evalops;
        evalops.parse_options.unknowns_enabled = false;
        evalops.allow_complex = false;
        evalops.structuring = STRUCTURING_SIMPLIFY;
        evalops.keep_zero_units = false;

        string expr_str(expression);
        string unlocalized_expr = calculator->unlocalizeExpression(expr_str, evalops.parse_options);
        PrintOptions printops = getPrintOptions(unlocalized_expr);
        string result = calculator->calculateAndPrint(unlocalized_expr, 2000, evalops, printops);

        return strdup_safe(result);
    }

    void free_result(char* result) {
        free(result);
    }

    char* get_all_functions() {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return nullptr;

        string out;
        for (size_t i = 0; i < calculator->functions.size(); i++) {
            MathFunction* func = calculator->functions[i];
            if (func && func->isActive()) {
                out += func->referenceName();
                out += '\t';
                out += func->category();
                out += '\n';
            }
        }
        return strdup_safe(out);
    }

    char* get_all_variables() {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return nullptr;

        string out;
        for (size_t i = 0; i < calculator->variables.size(); i++) {
            Variable* var = calculator->variables[i];
            if (var && var->isActive()) {
                out += var->referenceName();
                out += '\t';
                out += var->category();
                out += '\n';
            }
        }
        return strdup_safe(out);
    }
}
