package updater

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
	{Name: "Fit", Func: (*service).Fit},
	{Name: "Respond", Func: (*service).Respond},
	{Name: "Watch", Func: (*service).Watch},
}

func init() { mygo.BindMethods(reflect.TypeFor[*service](), serviceMethods...) }
