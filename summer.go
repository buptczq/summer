package summer

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"io/ioutil"
)

type xmlVapor struct {
	Name    string `xml:"name,attr"`
	Dew     string `xml:"dew,attr"`
	Value   string `xml:"value,attr"`
	Private bool   `xml:"private,attr"`
	Auto    bool   `xml:"auto,attr"`
}

type xmlDew struct {
	Id    string     `xml:"id,attr"`
	Class string     `xml:"class,attr"`
	Vapor []xmlVapor `xml:"vapor"`
}

type xmlRain struct {
	XMLName xml.Name `xml:"rain"`
	Dew     []xmlDew `xml:"dew"`
}

func setStructField(s interface{}, fieldName string, value string) error {
	if !isStructPtr(reflect.TypeOf(s)) {
		return fmt.Errorf("need a struct")
	}
	v := reflect.ValueOf(s).Elem().FieldByName(fieldName)
	kt := v.Type()
	if !v.CanSet() {
		return fmt.Errorf("field %s can't set", fieldName)
	}
	switch v.Kind() {
	case reflect.String:
		v.Set(reflect.ValueOf(value).Convert(kt))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(n).Convert(kt))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(n).Convert(kt))
	case reflect.Bool:
		n, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		v.SetBool(n)
	default:
		return fmt.Errorf("unsupport type %s", kt.String())
	}
	return nil
}

func (c *Container) XMLConfigurationContainer(data []byte, logger Logger) (*Graph, error) {
	var r xmlRain
	app := &Graph{Logger: logger}
	if err := xml.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	for _, d := range r.Dew {
		object := c.Get(d.Class)
		if object == nil {
			return nil, fmt.Errorf("dew %s#%s doesn't exist", d.Class, d.Id)
		}
		options := make(map[string]Option)
		for _, v := range d.Vapor {
			if v.Name == "" {
				return nil, fmt.Errorf("expected a vapor name at Dew %s#%s", d.Class, d.Id)
			}
			if v.Dew != "" {
				options[v.Name] = Option{v.Dew, v.Private}
			} else {
				if v.Auto {
					options[v.Name] = Option{"", v.Private}
				} else {
					if err := setStructField(object, v.Name, v.Value); err != nil {
						return nil, err
					}
				}
			}
		}
		err := app.Provide(&Dew{
			Value:   object,
			Name:    d.Id,
			Options: options,
		})
		if err != nil {
			return nil, err
		}
	}
	if err := app.Populate(); err != nil {
		return nil, err
	}
	return app, nil
}

func (c *Container) XMLFileConfigurationContainer(filename string, logger Logger) (*Graph, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return c.XMLConfigurationContainer(data, logger)
}
