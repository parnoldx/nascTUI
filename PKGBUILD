# Maintainer: parnoldx <nasc@pa.unbox.at>
pkgname=nasctui
pkgver=1.0.6
pkgrel=1
pkgdesc="Do maths like a normal person - A terminal calculator"
arch=('x86_64' 'i686' 'aarch64')
url="https://github.com/parnoldx/nascTUI"
license=('GPL2')
depends=('libqalculate')
makedepends=('go' 'gcc' 'pkgconf' 'git')
options=('!debug')
source=("git+https://github.com/parnoldx/nascTUI.git")
sha256sums=('SKIP')

pkgver() {
    cd "$srcdir/nascTUI"
    git describe --tags --abbrev=0 | sed 's/^v//'
}

prepare() {
    cd "$srcdir/nascTUI"
    git checkout "$(git describe --tags --abbrev=0)"
}

build() {
    cd "$srcdir/nascTUI"
    # cgo compiles src/calc_wrapper.cpp automatically
    local version=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
    cd src
    export CGO_ENABLED=1
    go build -trimpath -buildmode=pie -mod=readonly -modcacherw -ldflags "-X main.version=$version" -o ../nasc
}

package() {
    cd "$srcdir/nascTUI"
    install -Dm755 nasc "$pkgdir/usr/bin/nasc"
}
