package sqlite

import (
	"reflect"

	"github.com/egoist/mygo"
)

// serviceMethods are the methods of the plugin's service, which mygo build cannot
// name (the type is unexported, in this package) and so gives to Bind itself: an
// app built with the tag mygo_static binds the service from this table, not
// from reflection. TestServiceMethods fails when a method is added to the
// service and not here.
var serviceMethods = []mygo.BoundMethod{
	{Name: "Close", Func: (*service).Close},
	{Name: "Execute", Func: (*service).Execute},
	{Name: "Open", Func: (*service).Open},
	{Name: "Query", Func: (*service).Query},
	{Name: "Transaction", Func: (*service).Transaction},
}

func init() { mygo.BindMethods(reflect.TypeFor[*service](), serviceMethods...) }
