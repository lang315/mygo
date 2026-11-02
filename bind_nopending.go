//go:build !mygo_static && !js

package mygo

import "reflect"

// Without the tag mygo_static Bind lists the methods by reflection, which cannot
// lack a table: nothing waits for one (bind_pending.go).

func resolvePending(reflect.Type) {}

func checkPending() {}

func pendBind(error, *service, reflect.Value) (bool, error) { return false, nil }
