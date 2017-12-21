package summer

import (
	"fmt"
	"testing"
)

type StructAnswer struct {
	Ans   int
	List  []string
	Array [5]string
}

func (s *StructAnswer) Answer() int {
	return s.Ans
}

type AnswerSpeaker struct {
	Answer Answerable
}

func (s *AnswerSpeaker) Start() error {
	if s.Answer.Answer() != 666 {
		return fmt.Errorf("bad answer")
	}
	return nil
}

func TestContainer_XMLConfigurationContainer(t *testing.T) {
	con := new(Container)
	con.Register(AnswerSpeaker{})
	con.Register(StructAnswer{})
	config := []byte(`
<rain>
<dew id="test" class="summer.StructAnswer">
<vapor name="Ans"  value="666" />
</dew>
<dew id="checker" class="summer.AnswerSpeaker">
<vapor name="Answer"  dew="test" />
</dew>
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app.Start() != nil {
		t.Fail()
	}
	app.Stop()
}

func TestContainer_XMLConfigurationContainer_BadXML(t *testing.T) {
	con := new(Container)
	config := []byte(`
<rain>
<dew id="test" class="summer.StructAnswer">
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app != nil {
		t.Fail()
	}
}

func TestContainer_XMLConfigurationContainer_BadClass(t *testing.T) {
	con := new(Container)
	config := []byte(`
<rain>
<dew id="test" class="summer.StructAnswer" />
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app != nil {
		t.Fail()
	}
}

func TestContainer_XMLConfigurationContainer_UnnamedVapor(t *testing.T) {
	con := new(Container)
	con.Register(AnswerSpeaker{})
	con.Register(StructAnswer{})
	config := []byte(`
<rain>
<dew id="test" class="summer.StructAnswer">
<vapor />
</dew>
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app != nil {
		t.Fail()
	}
}

func TestContainer_BadStructFieldType(t *testing.T) {
	con := new(Container)
	con.Register(AnswerSpeaker{})
	con.Register(StructAnswer{})
	config := []byte(`
<rain>
<dew id="test" class="summer.StructAnswer">
<vapor name="Ans"  value="666" />
<vapor name="List"  value="test" />
</dew>
<dew id="server" class="summer.AnswerSpeaker">
<vapor name="Answer"  dew="test" />
</dew>
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app != nil {
		t.Fail()
	}
}

func TestContainer_DupName(t *testing.T) {
	con := new(Container)
	con.Register(AnswerSpeaker{})
	con.Register(StructAnswer{})
	config := []byte(`
<rain>
<dew id="test" class="summer.StructAnswer">
<vapor name="Ans"  value="666" />
</dew>
<dew id="server" class="summer.AnswerSpeaker">
<vapor name="Answer"  dew="test" />
</dew>
<dew id="server" class="summer.AnswerSpeaker">
<vapor name="Answer"  dew="test" />
</dew>
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app != nil {
		t.Fail()
	}
}

func TestContainer_UnknownId(t *testing.T) {
	con := new(Container)
	con.Register(AnswerSpeaker{})
	con.Register(StructAnswer{})
	config := []byte(`
<rain>
<dew id="server" class="summer.AnswerSpeaker">
<vapor name="Answer"  dew="test" />
</dew>
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app != nil {
		t.Fail()
	}
}

type test_struct struct {
	Int  int
	Str  string
	Uint uint
	Bool bool
	List []string
}

func TestContainer_SetField(t *testing.T) {
	var s test_struct
	setStructField(&s, "Int", "-213")
	setStructField(&s, "Str", "fdsaf")
	setStructField(&s, "Uint", "324324234")
	setStructField(&s, "Bool", "true")
	if s.Int != -213 || s.Str != "fdsaf" || s.Uint != 324324234 || s.Bool != true {
		t.Fail()
	}
}

func TestContainer_SetField_Bad(t *testing.T) {
	var s test_struct
	str := "123"
	if setStructField(&s, "Fake", "-213") == nil {
		t.Fail()
	}
	if setStructField(&str, "Int", "-213") == nil {
		t.Fail()
	}
	if str != "123" {
		t.Fail()
	}
	if setStructField(&s, "Int", "abc") == nil {
		t.Fail()
	}
	if setStructField(&s, "Uint", "-66") == nil {
		t.Fail()
	}
	if setStructField(&s, "Bool", "abc") == nil {
		t.Fail()
	}
}

func TestContainer_AutoInject(t *testing.T) {
	con := new(Container)
	con.Register(AnswerSpeaker{})
	con.Register(StructAnswer{})
	config := []byte(`
<rain>
<dew class="summer.StructAnswer">
<vapor name="Ans"  value="666" />
</dew>
<dew id="checker" class="summer.AnswerSpeaker">
<vapor name="Answer"  auto="True" />
</dew>
</rain>
`)
	app, _ := con.XMLConfigurationContainer(config, nil)
	if app.Start() != nil {
		t.Fail()
	}
	app.Stop()
}

type StructInlineTest struct {
	List  []string
	Array [5]int
	Map   map[string]string
}

func TestContainer_XMLInjectList(t *testing.T) {
	con := new(Container)
	con.Register(StructInlineTest{})
	config := []byte(`
<rain>
<dew id="test" class="summer.StructInlineTest">
<vapor name="List">
	<vapor value="test1" />
	<vapor value="test2" />
	<vapor value="test3" />
</vapor>
<vapor name="Array">
	<vapor value="1" />
	<vapor value="2" />
	<vapor value="3" />
	<vapor value="4" />
	<vapor value="5" />
</vapor>
<vapor name="Map">
	<vapor name="key1" value="test1" />
	<vapor name="key2" value="test2" />
</vapor>
</dew>
</rain>
`)
	con.XMLConfigurationContainer(config, nil)
	app, err := con.XMLConfigurationContainer(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	if app.Start() != nil {
		t.Fail()
	}
	test := app.GetDewByName("test").Value.(*StructInlineTest)
	result1 := []string{"test1", "test2", "test3"}
	result2 := []int{1, 2, 3, 4, 5}
	if test.Map["key1"] != "test1" || test.Map["key2"] != "test2" {
		t.Fail()
	}
	for i := range result1 {
		if test.List[i] != result1[i] {
			t.Fail()
		}
	}
	for i := range result2 {
		if test.Array[i] != result2[i] {
			t.Fail()
		}
	}
	app.Stop()
}

type StructUmarshalTestAns struct {
	A int
}

type StructUmarshalTest struct {
	A StructUmarshalTestAns
}

func (s *StructUmarshalTestAns) UnmarshalText(text []byte) error {
	if string(text) == "test" {
		s.A = 666
	}
	return nil
}

func (s *StructUmarshalTest) Answer() int {
	return s.A.A
}

func TestContainer_XMLUmarshalTest(t *testing.T) {
	con := new(Container)
	con.Register(AnswerSpeaker{})
	con.Register(StructUmarshalTest{})
	config := []byte(`
<rain>
<dew id="test" class="summer.StructUmarshalTest">
<vapor name="A"  value="test" />
</dew>
<dew id="checker" class="summer.AnswerSpeaker">
<vapor name="Answer"  dew="test" />
</dew>
</rain>
`)
	app, err := con.XMLConfigurationContainer(config, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	if app.Start() != nil {
		t.Fail()
	}
	app.Stop()
}
