package summer

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func init() {
	// we rely on math.Rand in Graph.Objects() and this gives it some randomness.
	rand.Seed(time.Now().UnixNano())
}

type Answerable interface {
	Answer() int
}

type TypeAnswerStruct struct {
	answer  int
	private int
}

func (t *TypeAnswerStruct) Answer() int {
	return t.answer
}

type TypeNestedStruct struct {
	A *TypeAnswerStruct
}

func (t *TypeNestedStruct) Answer() int {
	return t.A.Answer()
}

func TestRequireTag(t *testing.T) {
	var v struct {
		A *TypeAnswerStruct
		B *TypeNestedStruct
	}
	g := Graph{}
	g.Provide(&Dew{Value: &v, Options: map[string]Option{"B": Option{"", false}}})
	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.A != nil {
		t.Fatal("v.A is not nil")
	}
	if v.B == nil {
		t.Fatal("v.B is nil")
	}
}

type TypeWithNonPointerInject struct {
	A int
}

func TestErrorOnNonPointerInject(t *testing.T) {
	var a TypeWithNonPointerInject
	g := Graph{}
	g.Provide(&Dew{Value: &a, Options: map[string]Option{"A": Option{"", false}}})
	err := g.Populate()
	if err == nil {
		t.Fatalf("expected error for %+v", a)
	}
	const msg = "found inject option on unsupported field A in type *summer.TypeWithNonPointerInject"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeWithNonPointerStructInject struct {
	A *int
}

func TestErrorOnNonPointerStructInject(t *testing.T) {
	var a TypeWithNonPointerStructInject
	g := Graph{}
	g.Provide(&Dew{Value: &a, Options: map[string]Option{"A": Option{"", false}}})
	err := g.Populate()
	if err == nil {
		t.Fatalf("expected error for %+v", a)
	}

	const msg = "found inject option on unsupported field A in type *summer.TypeWithNonPointerStructInject"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestInjectSimple(t *testing.T) {
	var v struct {
		A *TypeAnswerStruct
		B *TypeNestedStruct
	}
	var b TypeNestedStruct

	g := Graph{}
	g.Provide(&Dew{Value: &v, Options: map[string]Option{
		"A": Option{"", false},
		"B": Option{"", false},
	}})
	g.Provide(&Dew{Value: &b, Options: map[string]Option{
		"A": Option{"", false},
	}})
	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.A == nil {
		t.Fatal("v.A is nil")
	}
	if v.B == nil {
		t.Fatal("v.B is nil")
	}
	if v.B.A == nil {
		t.Fatal("v.B.A is nil")
	}
	if v.A != v.B.A {
		t.Fatal("got different instances of A")
	}
}

func TestDoesNotOverwrite(t *testing.T) {
	a := &TypeAnswerStruct{}
	var v struct {
		A *TypeAnswerStruct
		B *TypeNestedStruct
	}
	v.A = a
	g := Graph{}
	g.Provide(&Dew{Value: &v, Options: map[string]Option{
		"A": Option{"", false},
		"B": Option{"", false},
	}})
	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.A != a {
		t.Fatal("original A was lost")
	}
	if v.B == nil {
		t.Fatal("v.B is nil")
	}
}

func TestPrivate(t *testing.T) {
	var v struct {
		A *TypeAnswerStruct
		B *TypeNestedStruct
	}

	var b TypeNestedStruct

	g := Graph{}
	g.Provide(&Dew{Value: &v, Options: map[string]Option{
		"A": Option{"", true},
		"B": Option{"", false},
	}})
	g.Provide(&Dew{Value: &b, Options: map[string]Option{
		"A": Option{"", false},
	}})
	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.A == nil {
		t.Fatal("v.A is nil")
	}
	if v.B == nil {
		t.Fatal("v.B is nil")
	}
	if v.B.A == nil {
		t.Fatal("v.B.A is nil")
	}
	if v.A == v.B.A {
		t.Fatal("got the same A")
	}
}

func TestProvideWithFields(t *testing.T) {
	var g Graph
	a := &TypeAnswerStruct{}
	err := g.Provide(&Dew{Value: &a, Fields: map[string]*Dew{}})
	if err == nil {
		t.Fatal("err is nil")
	}
	const msg = "fields were specified on object **summer.TypeAnswerStruct when it was provided"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestProvideNonPointer(t *testing.T) {
	var g Graph
	var i int
	err := g.Provide(&Dew{Value: i})
	if err == nil {
		t.Fatal("expected error")
	}

	const msg = "expected unnamed object value to be a pointer to a struct but got type int with value 0"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestProvideNonPointerStruct(t *testing.T) {
	var g Graph
	var i *int
	err := g.Provide(&Dew{Value: i})
	if err == nil {
		t.Fatal("expected error")
	}

	const msg = "expected unnamed object value to be a pointer to a struct but got type *int with value <nil>"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestProvideTwoOfTheSame(t *testing.T) {
	var g Graph
	a := TypeAnswerStruct{}
	err := g.Provide(&Dew{Value: &a})
	if err != nil {
		t.Fatal(err)
	}

	err = g.Provide(&Dew{Value: &a})
	if err == nil {
		t.Fatal("expected error")
	}

	const msg = "provided two unnamed instances of type *summer.TypeAnswerStruct"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestProvideTwoWithTheSameName(t *testing.T) {
	var g Graph
	const name = "foo"
	a := TypeAnswerStruct{}
	err := g.Provide(&Dew{Value: &a, Name: name})
	if err != nil {
		t.Fatal(err)
	}

	err = g.Provide(&Dew{Value: &a, Name: name})
	if err == nil {
		t.Fatal("expected error")
	}

	const msg = "provided two instances named foo"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestNamedInstanceWithDependencies(t *testing.T) {
	var g Graph
	a := &TypeNestedStruct{}
	if err := g.Provide(&Dew{
		Value: a,
		Name:  "foo", Options: map[string]Option{"A": Option{"", false}},
	}); err != nil {
		t.Fatal(err)
	}

	var c struct {
		A *TypeNestedStruct
	}
	if err := g.Provide(&Dew{
		Value:   &c,
		Options: map[string]Option{"A": Option{"foo", false}},
	}); err != nil {
		t.Fatal(err)
	}

	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}

	if c.A.A == nil {
		t.Fatal("c.A.A was not injected")
	}
}

type TypeWithMissingNamed struct {
	A *TypeAnswerStruct `inject:"foo"`
}

func TestTagWithMissingNamed(t *testing.T) {
	var g Graph
	var a TypeWithMissingNamed
	if err := g.Provide(&Dew{
		Value:   &a,
		Options: map[string]Option{"A": Option{"foo", false}},
	}); err != nil {
		t.Fatal(err)
	}
	err := g.Populate()
	if err == nil {
		t.Fatalf("expected error for %+v", a)
	}

	const msg = "did not find object named foo required by field A in type *summer.TypeWithMissingNamed"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestCompleteProvides(t *testing.T) {
	var g Graph
	var v struct {
		A *TypeAnswerStruct
	}

	if err := g.Provide(&Dew{
		Value:    &v,
		Options:  map[string]Option{"A": Option{"", false}},
		Complete: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.A != nil {
		t.Fatal("v.A was not nil")
	}
}

func TestCompleteNamedProvides(t *testing.T) {
	var g Graph
	var v struct {
		A *TypeAnswerStruct
	}

	if err := g.Provide(&Dew{
		Name:     "foo",
		Value:    &v,
		Options:  map[string]Option{"A": Option{"", false}},
		Complete: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.A != nil {
		t.Fatal("v.A was not nil")
	}
}

type TypeInjectInterfaceMissing struct {
	Answerable Answerable
}

func TestInjectInterfaceMissing(t *testing.T) {
	var v TypeInjectInterfaceMissing
	var g Graph
	if err := g.Provide(&Dew{
		Value:   &v,
		Options: map[string]Option{"Answerable": Option{"", false}},
	}); err != nil {
		t.Fatal(err)
	}

	err := g.Populate()
	if err == nil {
		t.Fatal("did not find expected error")
	}
	const msg = "found no assignable value for field Answerable in type *summer.TypeInjectInterfaceMissing"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeInjectInterface struct {
	Answerable Answerable
	A          *TypeAnswerStruct
}

func TestInjectInterface(t *testing.T) {
	var v TypeInjectInterface
	var g Graph
	if err := g.Provide(&Dew{
		Value: &v,
		Options: map[string]Option{
			"Answerable": Option{"", false},
			"A":          Option{"", false},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.Answerable == nil || v.Answerable != v.A {
		t.Fatalf(
			"expected the same but got Answerable = %T %+v / A = %T %+v",
			v.Answerable,
			v.Answerable,
			v.A,
			v.A,
		)
	}
}

type TypeWithInvalidNamedType struct {
	A *TypeNestedStruct `inject:"foo"`
}

func TestInvalidNamedInstanceType(t *testing.T) {
	var g Graph
	a := &TypeAnswerStruct{}
	if err := g.Provide(&Dew{Value: a, Name: "foo"}); err != nil {
		t.Fatal(err)
	}

	var c TypeWithInvalidNamedType
	if err := g.Provide(&Dew{
		Value: &c,
		Options: map[string]Option{
			"A": Option{"foo", false},
		},
	}); err != nil {
		t.Fatal(err)
	}
	err := g.Populate()
	if err == nil {
		t.Fatal("did not find expected error")
	}

	const msg = "object named foo of type *summer.TypeNestedStruct is not assignable to field A (*summer.TypeAnswerStruct) in type *summer.TypeWithInvalidNamedType"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeWithInjectOnPrivateField struct {
	a *TypeAnswerStruct
}

func TestInjectOnPrivateField(t *testing.T) {
	var a TypeWithInjectOnPrivateField
	var g Graph
	if err := g.Provide(&Dew{
		Value: &a,
		Options: map[string]Option{
			"a": Option{"", false},
		},
	}); err != nil {
		t.Fatal(err)
	}
	err := g.Populate()
	if err == nil {
		t.Fatal("did not find expected error")
	}

	const msg = "inject requested on unexported field a in type *summer.TypeWithInjectOnPrivateField"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeWithInjectOnPrivateInterfaceField struct {
	a Answerable
}

func TestInjectOnPrivateInterfaceField(t *testing.T) {
	var a TypeWithInjectOnPrivateField
	var g Graph
	if err := g.Provide(&Dew{
		Value: &a,
		Options: map[string]Option{
			"a": Option{"", false},
		},
	}); err != nil {
		t.Fatal(err)
	}

	err := g.Populate()
	if err == nil {
		t.Fatal("did not find expected error")
	}
	const msg = "inject requested on unexported field a in type *summer.TypeWithInjectOnPrivateField"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

func TestInjectNamedOnPrivateInterfaceField(t *testing.T) {
	var a TypeWithInjectOnPrivateField
	var g Graph
	if err := g.Provide(&Dew{
		Value: &a,
		Name:  "foo",
		Options: map[string]Option{
			"a": Option{"", false},
		},
	}); err != nil {
		t.Fatal(err)
	}

	err := g.Populate()
	if err == nil {
		t.Fatal("did not find expected error")
	}
	const msg = "inject requested on unexported field a in type *summer.TypeWithInjectOnPrivateField"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeInjectPrivateInterface struct {
	Answerable Answerable
	B          *TypeNestedStruct
}

func TestInjectPrivateInterface(t *testing.T) {
	var v TypeInjectPrivateInterface
	var g Graph
	if err := g.Provide(&Dew{
		Value: &v,
		Options: map[string]Option{
			"Answerable": Option{"", true},
			"B":          Option{"", true},
		},
	}); err != nil {
		t.Fatal(err)
	}
	err := g.Populate()
	if err == nil {
		t.Fatal("did not find expected error")
	}

	const msg = "found private inject option on interface field Answerable in type *summer.TypeInjectPrivateInterface"
	if err.Error() != msg {
		t.Fatalf("expected:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeInjectTwoSatisfyInterface struct {
	Answerable Answerable
	A          *TypeAnswerStruct
	B          *TypeNestedStruct
}

func TestInjectTwoSatisfyInterface(t *testing.T) {
	var v TypeInjectTwoSatisfyInterface
	var g Graph
	if err := g.Provide(&Dew{
		Value: &v,
		Options: map[string]Option{
			"Answerable": Option{"", false},
			"A":          Option{"", false},
			"B":          Option{"", false},
		},
	}); err != nil {
		t.Fatal(err)
	}
	err := g.Populate()

	const msg = "found two assignable values for field Answerable in type *summer.TypeInjectTwoSatisfyInterface. one type *summer.TypeAnswerStruct with value &{0 0} and another type *summer.TypeNestedStruct with value"
	if !strings.HasPrefix(err.Error(), msg) {
		t.Fatalf("expected prefix:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeInjectNamedTwoSatisfyInterface struct {
	Answerable Answerable
	A          *TypeAnswerStruct
	B          *TypeNestedStruct
}

func TestInjectNamedTwoSatisfyInterface(t *testing.T) {
	var g Graph
	var v TypeInjectNamedTwoSatisfyInterface
	if err := g.Provide(&Dew{
		Value: &v,
		Name:  "foo",
		Options: map[string]Option{
			"Answerable": Option{"", false},
			"A":          Option{"", false},
			"B":          Option{"", false},
		},
	}); err != nil {
		t.Fatal(err)
	}

	err := g.Populate()
	if err == nil {
		t.Fatal("was expecting error")
	}

	const msg = "found two assignable values for field Answerable in type *summer.TypeInjectNamedTwoSatisfyInterface. one type *summer.TypeAnswerStruct with value &{0 0} and another type *summer.TypeNestedStruct with value"
	if !strings.HasPrefix(err.Error(), msg) {
		t.Fatalf("expected prefix:\n%s\nactual:\n%s", msg, err.Error())
	}
}

type TypeWithNonPointerNamedInject struct {
	A int
}

func TestErrorOnNonPointerNamedInject(t *testing.T) {
	var g Graph
	if err := g.Provide(&Dew{Name: "foo", Value: 42}); err != nil {
		t.Fatal(err)
	}

	var v TypeWithNonPointerNamedInject
	if err := g.Provide(&Dew{
		Value: &v,
		Options: map[string]Option{
			"A": Option{"foo", false},
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}

	if v.A != 42 {
		t.Fatalf("expected v.A = 42 but got %d", v.A)
	}
}

func TestInjectMap(t *testing.T) {
	var g Graph
	var v struct {
		A map[string]int
	}
	if err := g.Provide(&Dew{
		Value: &v,
		Options: map[string]Option{
			"A": Option{"", true},
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := g.Populate(); err != nil {
		t.Fatal(err)
	}
	if v.A == nil {
		t.Fatal("v.A is nil")
	}
}
