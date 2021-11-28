package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	sigsyaml "sigs.k8s.io/yaml"
)

func resourceKey(gvk *schema.GroupVersionKind, o metav1.Object) (string, string, string) {
	group := "-"
	namespace := "-"
	name := o.GetName()
	if gvk.Group != "" {
		group = gvk.Group
	}
	if o.GetNamespace() != "" {
		namespace = o.GetNamespace()
	}
	if name == "" {
		name = o.GetGenerateName()
	}
	return strings.Join([]string{group, gvk.Version, gvk.Kind, namespace, name}, "/"), o.GetNamespace(), name
}

type Info struct {
	*schema.GroupVersionKind
	Namespace string
	Name      string
}

func normalize(o runtime.Object, dec runtime.Serializer) (string, error) {
	buf := new(bytes.Buffer)
	if err := dec.Encode(o, buf); err != nil {
		return "", err
	}
	if b, err := sigsyaml.JSONToYAML(buf.Bytes()); err != nil {
		return "", err
	} else {
		return string(b), nil
	}
}

func loadObjects(p string, gvks map[string]Info, n bool) map[string]string {
	data, err := ioutil.ReadFile(p)
	if err != nil {
		panic(err)
	}
	objects := map[string]string{}
	for _, part := range strings.Split(string(data), "---") {
		part = strings.TrimSpace(part) + "\n"
		if len(part) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{}
		dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		runtimeObject, gvk, err := dec.Decode([]byte(part), nil, obj)
		if err != nil {
			log.Print(part, len(part))
			panic(err)
		}
		metaObject, ok := runtimeObject.(metav1.Object)
		if !ok {
			panic("Failed to convert from runtime.Object to meta.Object")
		}
		if n {
			p, err := normalize(runtimeObject, dec)
			if err != nil {
				panic(err)
			}
			part = p
		}
		key, namespace, name := resourceKey(gvk, metaObject)
		objects[key] = part
		gvks[key] = Info{
			GroupVersionKind: gvk,
			Namespace:        namespace,
			Name:             name,
		}
	}
	return objects
}

var report = `
{{- define "yesno" }}
  {{- if . }}:white_check_mark:{{ else }}:red_circle:{{ end }}
{{- end }}

{{- define "difflink" }}
  {{- $diff := ne .Source .Target }}
  {{- if $diff -}}
[^{{ if .Info.Group }}{{ replaceAll .Info.Group "." "" }}--{{ end }}{{ .Info.Version }}--{{ .Info.Kind | toLower }}--{{ if .Info.Namespace }}{{ .Info.Namespace }}--{{ end }}{{ .Info.Name }}]
  {{- end }}
{{- end }}

{{- $empty := true }}
{{- $hasDiff := false }}
{{- range .Resources }}
  {{- if ne .Source .Target }}
    {{- $hasDiff = true -}}
  {{ end }}
  {{- if or $hasDiff (eq $.Mode "full") }}
    {{- $empty = false -}}
  {{ end }}
{{- end -}}
# Changes report [{{ .Mode }}]{{ if not $empty }} [^legend]{{ end }}

{{- if $empty }}

No resources to show in this report
{{- else }}

| Kind (Version) | Namespace | Name | B | A | :eyes: |
|---|---|---|:-:|:-:|---|
  {{- range .Resources }}
    {{- $diff := ne .Source .Target }}
    {{- if or $diff (eq $.Mode "full") }}
| {{ .Info.Kind }} ({{ .Info.Version }}) | {{ .Info.Namespace | code }} | {{ .Info.Name | code }} | {{ template "yesno" .InSource }} | {{ template "yesno" .InTarget }} | {{ template "difflink" . }} |
    {{- end }}
  {{- end }}

[^legend]:
    ### Legend
    - **B :** *Before*
    - **A :** *After*

  {{- if $hasDiff }}
    {{- range .Resources }}
      {{- $diff := ne .Source .Target }}
      {{- if $diff }}
[^{{ if .Info.Group }}{{ replaceAll .Info.Group "." "" }}--{{ end }}{{ .Info.Version }}--{{ .Info.Kind | toLower }}--{{ if .Info.Namespace }}{{ .Info.Namespace }}--{{ end }}{{ .Info.Name }}]:
    ### {{ if .Info.Group }}{{ .Info.Group }} / {{ end }}{{ .Info.Version }} / {{ .Info.Kind }} / {{ if .Info.Namespace }}{{ .Info.Namespace }} / {{ end }}{{ .Info.Name }}
{{ diff .Source .Target | indent 4 }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
`

type ResourceInfo struct {
	Info     Info
	InSource bool
	InTarget bool
	Source   string
	Target   string
}

type Diff struct {
	Mode      string
	Source    string
	Target    string
	Resources []ResourceInfo
}

func main() {
	var source = flag.String("source", "", "source file path")
	var target = flag.String("target", "", "target file path")
	var mode = flag.String("mode", "full", "report mode (full or diff)")
	var normalize = flag.Bool("normalize", false, "normalize yaml before diff")
	flag.Parse()
	if *mode != "full" && *target == "diff" {
		panic("mode must be full or diff")
	}
	if *source == "" && *target == "" {
		panic("source and target are required")
	}
	gvks := map[string]Info{}
	sourceObjects := loadObjects(*source, gvks, *normalize)
	targetObjects := loadObjects(*target, gvks, *normalize)
	keys := make([]string, 0)
	for k := range gvks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var infos []ResourceInfo
	for _, k := range keys {
		source, inSource := sourceObjects[k]
		target, inTarget := targetObjects[k]
		infos = append(infos, ResourceInfo{
			Info:     gvks[k],
			InSource: inSource,
			InTarget: inTarget,
			Source:   source,
			Target:   target,
		})
	}
	tmpl := template.New("test")
	tmpl = tmpl.Funcs(sprig.TxtFuncMap())
	tmpl = tmpl.Funcs(template.FuncMap{
		"toLower":    strings.ToLower,
		"replaceAll": strings.ReplaceAll,
		"code": func(s string) string {
			if s == "" {
				return s
			}
			return "`" + s + "`"
		},
		"diff": func(s, t string) string {
			edits := myers.ComputeEdits(span.URIFromPath("source"), s, t)
			return "```diff\n" + fmt.Sprint(gotextdiff.ToUnified("source", "target", s, edits)) + "```"
		},
	})
	tmpl, err := tmpl.Parse(report)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, Diff{
		Mode:      *mode,
		Source:    *source,
		Target:    *target,
		Resources: infos,
	})
	if err != nil {
		panic(err)
	}
}
