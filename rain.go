package summer

import "reflect"

type Container struct {
	rain map[string]reflect.Type
}

func (c *Container) Register(proto interface{}) {
	if c.rain == nil {
		c.rain = make(map[string]reflect.Type)
	}
	reflectType := reflect.TypeOf(proto)
	c.rain[reflectType.String()] = reflectType
}

func (c *Container) GetType(name string) reflect.Type {
	if c.rain == nil {
		return nil
	}
	return c.rain[name]
}

func (c *Container) Get(name string) interface{} {
	reflectType := c.GetType(name)
	if reflectType == nil {
		return nil
	}
	return reflect.New(reflectType).Interface()
}

func (c *Container) GetMap() map[string]reflect.Type {
	return c.rain
}
