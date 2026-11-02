//go:build mygo_static

package sqlite

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/internal/bindcheck"
)

func TestServiceMethods(t *testing.T) {
	bindcheck.Table(t, reflect.TypeFor[*service](), serviceMethods)
}
