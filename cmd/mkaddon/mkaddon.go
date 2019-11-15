package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v2"
)

// AddonData represents top-level addon information.
type AddonData struct {
	// Addon is the name of the addon. This must match the addon package name.
	Addon string
	// Import is the import path of the addon.
	Import string
	// Protos is the list of protos this addon provides.
	Protos []ProtoData
	// Depends is the list of addons on which this addon depends. Each addon
	// listed is imported before any of the scripts in Scripts execute.
	Depends []string
	// Install is a list of proto data to add to existing Core or Addon protos.
	// These are added before any of the scripts in Scripts execute.
	Install []ProtoData
	// Scripts is the list of Io script files that should be executed after
	// initializing the protos but before completing the import. The text of
	// each script is copied into the generated plugin source code.
	Scripts []string
}

// ProtoData represents the data of a single addon proto.
type ProtoData struct {
	// Proto is the name of the proto.
	Proto string
	// Tag is the default text to use for the Tag argument to
	// vm.NewCFunction. If empty, the default is nil. This is also used as the
	// Tag value for the proto installed on the VM.
	Tag string
	// CFunctions, Strings, Numbers, and Bools give the proto's slots with
	// values of type CFunction, ImmutableSequence encoded in UTF-8, or Number,
	// or one of the boolean singletons true, false, and nil.
	Functions map[string]CFuncData
	Strings   map[string]string
	Numbers   map[string]float64
	Bools     map[string]string
	// Custom contains slots other than CFunctions, strings, and numbers, whose
	// values can and should be set up at compile time. Each slot is of the
	// form
	//
	//	slot: <literal Go code>
	//  e.g.
	//  readBuffer: 'vm.NewSequence([]byte{}, true, "latin1")'
	Custom map[string]string
	// Value is the text to use for the value of the object installed on the
	// VM.
	Value string
}

// CFuncData represents the data of a single CFunction slot.
type CFuncData struct {
	// Fn is the name of the Go function to use.
	Fn string
	// Tag, if non-empty, overrides the DefaultTag value of the ProtoData to
	// which this CFunction belongs.
	Tag string
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		fail(os.Args[0], "addon.yaml output.go")
	}
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fail(err)
	}
	var data AddonData
	if err = yaml.Unmarshal(b, &data); err != nil {
		fail(err)
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		fail(err)
	}
	if err = body.Execute(out, data); err != nil {
		fail(err)
	}
	if _, err = fmt.Fprintf(out, "var addon%sFiles = %#v\n\n", data.Addon, data.Scripts); err != nil {
		panic(err)
	}
	if _, err = fmt.Fprintf(out, "var addon%sIo = [][]byte{\n", data.Addon); err != nil {
		panic(err)
	}
	for _, i := range data.Scripts {
		buf := bytes.Buffer{}
		in, err := os.Open(i)
		if err != nil {
			panic(err)
		}
		w := zlib.NewWriter(&buf)
		if _, err = io.Copy(w, in); err != nil {
			panic(err)
		}
		w.Close()
		if _, err = fmt.Fprintf(out, "\t%#v,\n", buf.Bytes()); err != nil {
			panic(err)
		}
	}
	if _, err = fmt.Fprintln(out, "}"); err != nil {
		panic(err)
	}
	out.Close()
	if stat, err := os.Stat(data.Addon); err == nil {
		if !stat.IsDir() {
			panic("path " + data.Addon + " already exists and is not a directory")
		}
	} else if !os.IsNotExist(err) {
		panic(err)
	}
	if err = os.Mkdir(data.Addon, 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}
	out, err = os.Create(filepath.Join(data.Addon, "main.go"))
	if err != nil {
		panic(err)
	}
	if err = plugin.Execute(out, data); err != nil {
		panic(err)
	}
	out.Close()
}

var body = template.Must(template.New("body").Parse(source))

const source = `
{{- define "mkslots"}}iolang.Slots{
	{{range $k, $v := .Functions}}		{{printf "%q" $k}}: vm.NewCFunction({{$v.Fn}}, {{or $v.Tag .Tag "nil"}}),
	{{end}}{{range $k, $v := .Strings}}
			{{printf "%q" $k}}: vm.NewString({{printf "%q" $v}}),
	{{end}}{{range $k, $v := .Numbers}}
			{{printf "%q" $k}}: vm.NewNumber({{printf "%.17g" $v}}),
	{{end}}{{range $k, $v := .Bools}}
			{{printf "%q" $k}}: {{if eq $v "true" "True" "on" "1"}}vm.True{{else if eq $v "false" "False" "off" "0"}}vm.False{{else}}vm.Nil{{end}},
	{{end}}{{range $k, $v := .Custom}}
			{{printf "%q" $k}}: {{$v}},
	{{end}}	}
{{- end -}}
package {{.Addon}}

// Code generated by mkaddon; DO NOT EDIT

import (
	"bytes"
	"compress/zlib"

	"github.com/zephyrtronium/iolang"
)

// IoAddon returns a loader for the {{.Addon}} addon.
func IoAddon() iolang.Addon {
	return addon{{.Addon}}{}
}

type addon{{.Addon}} struct{}

func (addon{{.Addon}}) Name() string {
	return {{printf "%q" .Addon}}
}

var addon{{.Addon}}Protos = []string{
{{range .Protos}}	{{printf "%q" .Proto}},
{{end -}} }

func (addon{{.Addon}}) Protos() []string {
	return addon{{.Addon}}Protos
}

var addon{{.Addon}}Depends {{with .Depends}}= []string{
{{range .}}	{{printf "%q" .}}
{{end -}} }{{else}}[]string{{end}}

func (addon{{.Addon}}) Depends() []string {
	return addon{{.Addon}}Depends
}

func (addon{{.Addon}}) Init(vm *iolang.VM) {
{{range .Protos}}	slots{{.Proto}} := {{template "mkslots" .}}
	vm.Install({{printf "%q" .Proto}}, vm.ObjectWith(slots{{.Proto}}, []*iolang.Object{vm.BaseObject}, {{or .Value "nil"}}, {{or .Tag "nil"}}))
{{end}}
{{range .Install}}	merge{{.Proto}} := {{template "mkslots" .}}
	if obj, ok := vm.Core.GetLocalSlot({{printf "%q" .Proto}}); ok {
		obj.SetSlots(merge{{.Proto}})
	} else if obj, ok = vm.Addons.GetLocalSlot({{printf "%q" .Proto}}); ok {
		obj.SetSlots(merge{{.Proto}})
	} else {
		panic("cannot merge new slots onto {{.Proto}}")
	}
{{end}}
	for i, b := range addon{{.Addon}}Io {
		r, err := zlib.NewReader(bytes.NewReader(b))
		if err != nil {
			panic(err)
		}
		exc, stop := vm.DoReader(r, addon{{.Addon}}Files[i])
		if stop == iolang.ExceptionStop {
			panic(exc)
		}
	}
}

`

var plugin = template.Must(template.New("plugin").Parse(pluginsource))

const pluginsource = `package main

import (
	"github.com/zephyrtronium/iolang"
	{{printf "%q" .Import}}
)

// IoAddon returns an object to load the addon.
func IoAddon() iolang.Addon {
	return {{.Addon}}.IoAddon()
}
`
