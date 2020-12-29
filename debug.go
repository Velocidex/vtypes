package vtypes

import (
	"encoding/json"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"www.velocidex.com/golang/vfilter"
)

func Debug(arg interface{}) {
	spew.Dump(arg)
}

func JsonDump(v interface{}) {
	fmt.Println(StringIndent(v))
}

func StringIndent(v interface{}) string {
	result, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		panic(err)
	}
	return string(result)
}

func ScopeDebug(scope *vfilter.Scope, fmt string, args ...interface{}) {
	value, pres := scope.Resolve("DEBUG_VTYPES")
	if pres && scope.Bool(value) {
		scope.Log(fmt, args...)
	}
}
