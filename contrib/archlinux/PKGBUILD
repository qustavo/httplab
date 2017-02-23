# Maintainer: Gustavo Chain <gchain@gmail.com>
pkgname=httplab
pkgver=0.1
pkgrel=1
pkgdesc="An interactive web server"
arch=(any)
url="http://github.com/gchaincl/httplab"
license=('MIT')
makedepends=('go>=1.7')
provides=('httplab=$pkgver')
conflicts=('httplab')
replaces=('httplab')
install=
source=("$pkgname"::'git+https://github.com/gchaincl/httplab.git')
md5sums=(SKIP)

build() {
	cd "$pkgname"
	go build
}

package() {
	cd "$pkgname"
	install -D -s -m644 httplab "${pkgdir}/usr/bin/httplab"
}
