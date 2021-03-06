package summer

import (
	"encoding"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"os"
	"github.com/joho/godotenv"

)

var UMARSHALTEXT_TYPE = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

type xmlSubVapor struct {
	Name  string `xml:"name,attr"`
	Dew   string `xml:"dew,attr"`
	Value string `xml:"value,attr"`
}

type xmlVapor struct {
	Name    string        `xml:"name,attr"`
	Dew     string        `xml:"dew,attr"`
	Value   string        `xml:"value,attr"`
	Private bool          `xml:"private,attr"`
	Auto    bool          `xml:"auto,attr"`
	List    []xmlSubVapor `xml:"vapor"`
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

func setFieldWithString(v reflect.Value, value string) error {
	if !v.IsValid() {
		return fmt.Errorf("invalid field with value %s", value)
	}
	kt := v.Type()
	if !v.CanSet() {
		return fmt.Errorf("field of type %s can't set", kt.String())
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
	case reflect.Struct:
		if reflect.PtrTo(kt).Implements(UMARSHALTEXT_TYPE) {
			fv := reflect.New(kt)
			if itm, ok := fv.Interface().(encoding.TextUnmarshaler); ok {
				if err := itm.UnmarshalText([]byte(value)); err != nil {
					return err
				}
				v.Set(fv.Elem())
				return nil
			}
		}
		return fmt.Errorf("invalid inject %s into type %s", value, kt.String())
	default:
		return fmt.Errorf("invalid inject %s into type %s", value, kt.String())
	}
	return nil
}

func setStructField(s interface{}, fieldName string, value string) error {
	if !isStructPtr(reflect.TypeOf(s)) {
		return fmt.Errorf("need a struct")
	}
	v := reflect.ValueOf(s).Elem().FieldByName(fieldName)
	return setFieldWithString(v, value)
}

func setStructInlineField(s interface{}, fieldName string, list []xmlSubVapor) error {
	if !isStructPtr(reflect.TypeOf(s)) {
		return fmt.Errorf("need a struct")
	}
	v := reflect.ValueOf(s).Elem().FieldByName(fieldName)
	kt := v.Type()
	if !v.IsValid() {
		return fmt.Errorf("invalid field %s", fieldName)
	}
	if !v.CanSet() {
		return fmt.Errorf("field %s can't set", fieldName)
	}
	switch v.Kind() {
	case reflect.Slice:
		l := reflect.MakeSlice(kt, len(list), len(list))
		for i := range list {
			if err := setFieldWithString(l.Index(i), list[i].Value); err != nil {
				return err
			}
		}
		v.Set(l)
	case reflect.Array:
		if v.Len() != len(list) {
			return fmt.Errorf("the length of %s doesn't match array %s", fieldName, kt.String())
		}
		for i := range list {
			if err := setFieldWithString(v.Index(i), list[i].Value); err != nil {
				return err
			}
		}
	case reflect.Map:
		l := reflect.MakeMap(kt)
		for i := range list {
			valueKey := reflect.New(kt.Key()).Elem()
			valueVal := reflect.New(kt.Elem()).Elem()
			if err := setFieldWithString(valueKey, list[i].Name); err != nil {
				return err
			}
			if err := setFieldWithString(valueVal, list[i].Value); err != nil {
				return err
			}
			l.SetMapIndex(valueKey, valueVal)
		}
		v.Set(l)
	default:
		return fmt.Errorf("unsupported type %s", kt.String())

	}
	return nil
}

func (c *Container) XMLConfigurationContainer(data []byte, logger Logger) (*Graph, error) {
	var r xmlRain
	godotenv.Load()
	app := &Graph{Logger: logger}
	debug := func(f string, args ...interface{}) {
		if logger != nil {
			logger.Debugf(f, args...)
		}
	}
	if err := xml.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	for _, d := range r.Dew {
		// Instantiate objects
		object := c.Get(d.Class)
		oType := c.GetType(d.Class)
		if object == nil {
			return nil, fmt.Errorf("dew %s#%s doesn't exist", d.Class, d.Id)
		}
		options := make(map[string]Option)
		// Tag
		for i := 0; i < oType.NumField(); i++ {
			found, value, err := extract("vapor", string(oType.Field(i).Tag))
			if err != nil {
				continue
			}
			if !found {
				continue
			}
			switch value {
			case "auto":
				options[oType.Field(i).Name] = Option{Name: ""}
			}
		}
		// Vapor config from XML
		for _, v := range d.Vapor {
			// Inject arguments
			if v.Name == "" {
				return nil, fmt.Errorf("expected a vapor name at dew %s#%s", d.Class, d.Id)
			}
			if v.Dew != "" {
				if len(v.List) != 0 {
					return nil, fmt.Errorf("dew %s#%s shouldn't be a list or a map", d.Class, d.Id)
				}
				// Inject a named dew
				options[v.Name] = Option{Name: v.Dew}
			} else {
				if v.Auto {
					if len(v.List) != 0 {
						return nil, fmt.Errorf("auto vapor at dew %s#%s shouldn't be a list or a map", d.Class, d.Id)
					}
					// Inject a unnamed dew
					options[v.Name] = Option{Name: ""}
				} else {
					if len(v.List) == 0 {
						// Inject const value
						value := v.Value
						if os.Getenv(d.Class+"."+v.Name) != "" {
							value = os.Getenv(d.Class+"."+v.Name)
						}
						if err := setStructField(object, v.Name, value); err != nil {
							return nil, err
						}
						debug(
							"assigned %s to field %s in %s",
							v.Value,
							v.Name,
							d.Class,
						)
					} else {
						if v.List[0].Dew == "" {
							// Inject const list/map
							if err := setStructInlineField(object, v.Name, v.List); err != nil {
								return nil, err
							}
							debug(
								"assigned %s to field %s in %s",
								v.List,
								v.Name,
								d.Class,
							)
						} else {
							// Inject dew
							vaporOp := make([]VaporOption, len(v.List))
							for i := range v.List {
								vaporOp[i].Name = v.List[i].Name
								vaporOp[i].Dew = v.List[i].Dew
							}
							options[v.Name] = Option{Name: "", Vapor: vaporOp}
						}
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
