package summer

import (
	"testing"
)

var test_answer int

type AnswerSpeaker struct {
	Answer Answerable
}

type StructAnswer struct {
	Ans  int
	List []string
}

func (s *StructAnswer) Answer() int {
	return s.Ans
}

func (s *AnswerSpeaker) Start() error {
	test_answer = s.Answer.Answer()
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
	test_answer = 0
	app.Start()
	if test_answer != 666 {
		t.Failed()
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
		t.Failed()
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
		t.Failed()
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
		t.Failed()
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
		t.Failed()
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
		t.Failed()
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
		t.Failed()
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
		t.Failed()
	}
}

func TestContainer_SetField_Bad(t *testing.T) {
	var s test_struct
	str := "123"
	if setStructField(&str, "Int", "-213") != nil {
		t.Failed()
	}
	if str != "123" {
		t.Failed()
	}
	if setStructField(&s, "Int", "abc") != nil {
		t.Failed()
	}
	if setStructField(&s, "Uint", "-66") != nil {
		t.Failed()
	}
	if setStructField(&s, "Bool", "abc") != nil {
		t.Failed()
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
	test_answer = 0
	app.Start()
	if test_answer != 666 {
		t.Failed()
	}
	app.Stop()
}
