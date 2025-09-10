#include <string>
#include <libqalculate/Calculator.h>
#include <libqalculate/MathStructure.h>
#include <libqalculate/Function.h>
#include <libqalculate/Variable.h>
#include <stdlib.h>
#include <string.h>
#include <locale.h>
#include <mutex>

using namespace std;

static bool calculator_initialized = false;
static std::mutex calculator_mutex;

static void load_currencies() {
    if (calculator) {
        calculator->loadExchangeRates();
    }
}

static bool update_exchange_rates() {
    if (!calculator) return false;
    
    // Check if exchange rates are used
    int rates_used = calculator->exchangeRatesUsed();
    if (rates_used == 0) return false;
    
    // Check if rates need updating (7 days threshold)
    if (!calculator->checkExchangeRatesDate(7, false, true, rates_used)) {
        return false; // Rates are recent enough
    }
    
    // Fetch new exchange rates (15 second timeout)
    bool success = calculator->fetchExchangeRates(15, rates_used);
    if (success) {
        calculator->loadExchangeRates();
    }
    
    return success;
}

static void initialize_calculator() {
    std::lock_guard<std::mutex> lock(calculator_mutex);
    if (calculator_initialized) return;
    
    // Set system locale like Nasc
    setlocale(LC_ALL, "");
    
    // Initialize calculator using the global instance (libqalculate provides this)
    if (!calculator) {
        calculator = new Calculator();
    }
    
    // Load definitions like Nasc
    calculator->loadGlobalDefinitions();
    calculator->loadLocalDefinitions();
    load_currencies();
    
    // Configure evaluation options (like Nasc)
    calculator->useIntervalArithmetic(false);
    
    calculator_initialized = true;
}

extern "C" {
    void abort_calculation() {
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (calculator_initialized && calculator) {
            calculator->abort();
        }
    }

    bool update_exchange_rates_if_needed() {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) {
            return false;
        }
        
        return update_exchange_rates();
    }

    char* calculate_expression(const char* expression) {
        initialize_calculator();
        
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) {
            char* c_result = (char*)malloc(6);  // "Error" + null terminator
            strcpy(c_result, "Error");
            return c_result;
        }
        
        // Configure evaluation options exactly like Nasc
        EvaluationOptions evalops;
        evalops.parse_options.unknowns_enabled = false;
        evalops.allow_complex = false;
        evalops.structuring = STRUCTURING_SIMPLIFY;
        evalops.keep_zero_units = false;
        
        // Configure print options exactly like Nasc
        PrintOptions printops;
        printops.multiplication_sign = MULTIPLICATION_SIGN_ASTERISK;
        printops.number_fraction_format = FRACTION_DECIMAL;
        printops.max_decimals = 9;
        printops.use_max_decimals = true;
        printops.use_unicode_signs = true;
        printops.use_unit_prefixes = false;
        
        // Calculate the expression (preprocessing/postprocessing done in Go)
        string expr_str(expression);
        string result = calculator->calculateAndPrint(expr_str, 2000, evalops, printops);
        
        char* c_result = (char*)malloc(result.length() + 1);
        strcpy(c_result, result.c_str());
        return c_result;
    }

    void free_result(char* result) {
        free(result);
    }
    
    int get_function_count() {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return 0;
        
        int count = 0;
        for (size_t i = 0; i < calculator->functions.size(); i++) {
            MathFunction* func = calculator->functions[i];
            if (func && func->isActive()) {
                count++;
            }
        }
        return count;
    }
    
    char* get_function_name(int index) {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return nullptr;
        
        int activeIndex = 0;
        for (size_t i = 0; i < calculator->functions.size(); i++) {
            MathFunction* func = calculator->functions[i];
            if (func && func->isActive()) {
                if (activeIndex == index) {
                    string name = func->referenceName();
                    char* c_name = (char*)malloc(name.length() + 1);
                    strcpy(c_name, name.c_str());
                    return c_name;
                }
                activeIndex++;
            }
        }
        return nullptr;
    }
    
    int get_variable_count() {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return 0;
        
        int count = 0;
        for (size_t i = 0; i < calculator->variables.size(); i++) {
            Variable* var = calculator->variables[i];
            if (var && var->isActive()) {
                count++;
            }
        }
        return count;
    }
    
    char* get_variable_name(int index) {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return nullptr;
        
        int activeIndex = 0;
        for (size_t i = 0; i < calculator->variables.size(); i++) {
            Variable* var = calculator->variables[i];
            if (var && var->isActive()) {
                if (activeIndex == index) {
                    string name = var->referenceName();
                    char* c_name = (char*)malloc(name.length() + 1);
                    strcpy(c_name, name.c_str());
                    return c_name;
                }
                activeIndex++;
            }
        }
        return nullptr;
    }
    
    char* get_function_category(int index) {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return nullptr;
        
        int activeIndex = 0;
        for (size_t i = 0; i < calculator->functions.size(); i++) {
            MathFunction* func = calculator->functions[i];
            if (func && func->isActive()) {
                if (activeIndex == index) {
                    string category = func->category();
                    char* c_category = (char*)malloc(category.length() + 1);
                    strcpy(c_category, category.c_str());
                    return c_category;
                }
                activeIndex++;
            }
        }
        return nullptr;
    }
    
    char* get_variable_category(int index) {
        initialize_calculator();
        std::lock_guard<std::mutex> lock(calculator_mutex);
        if (!calculator_initialized || !calculator) return nullptr;
        
        int activeIndex = 0;
        for (size_t i = 0; i < calculator->variables.size(); i++) {
            Variable* var = calculator->variables[i];
            if (var && var->isActive()) {
                if (activeIndex == index) {
                    string category = var->category();
                    char* c_category = (char*)malloc(category.length() + 1);
                    strcpy(c_category, category.c_str());
                    return c_category;
                }
                activeIndex++;
            }
        }
        return nullptr;
    }
}