# Maintainer: Gustavo Chain <gchain@gmail.com>
pkgname=httplab
pkgver=0.2.1
pkgrel=2
pkgdesc="An interactive web server"
arch=(x86_64)
url="http://github.com/gchaincl/httplab"
license=('MIT')
makedepends=('wget')
provides=('httplab=$pkgver')
conflicts=('httplab')
replaces=('httplab')
install=
source=("$pkgname"::'https://github.com/gchaincl/httplab/releases/download/v0.2.1/httplab_linux_amd64')
md5sums=(
	'6e1051b464963eb40e89a786ef9dcce8'
)

package() {
	install -D -s -m755 "httplab" "${pkgdir}/usr/bin/${pkgname}"
}
