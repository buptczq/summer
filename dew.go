// Fork from github.com/facebookgo/inject
// By BUPTCZQ
package summer

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
)

// Logger allows for simple logging as inject traverses and populates the
// object graph.
type Logger interface {
	Debugf(f string, args ...interface{})
	Errorf(f string, args ...interface{})
}

// Vapor option
type VaporOption struct {
	Name string
	Dew  string
}

// Field option
type Option struct {
	Name  string
	Vapor []VaporOption
}

// Dependence type
type Dependence struct {
	Field  string
	Object *Dew
}

// An Dew in the Graph.
type Dew struct {
	Value        interface{}
	Name         string            // Optional
	Complete     bool              // If true, the Value will be considered complete
	Options      map[string]Option // The field names that named dependency were injected into
	Dependencies []*Dependence     // Dew's Dependencies
	reflectType  reflect.Type
	reflectValue reflect.Value
	created      bool // If true, the Dew was created by us
}

// String representation suitable for human consumption.
func (o *Dew) String() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, o.reflectType)
	if o.Name != "" {
		fmt.Fprintf(&buf, " named %s", o.Name)
	}
	return buf.String()
}

func (o *Dew) addDep(field string, dep *Dew) {
	if o.Dependencies == nil {
		o.Dependencies = make([]*Dependence, 0)
	}
	o.Dependencies = append(o.Dependencies, &Dependence{field, dep})
}

// The Graph of Objects.
type Graph struct {
	Logger      Logger // Optional, will trigger debug logging.
	unnamed     []*Dew
	unnamedType map[reflect.Type]bool
	named       map[string]*Dew
	started     []*Dew
}

// Provide objects to the Graph. The Dew documentation describes
// the impact of various fields.
func (g *Graph) Provide(objects ...*Dew) error {
	for _, o := range objects {
		o.reflectType = reflect.TypeOf(o.Value)
		o.reflectValue = reflect.ValueOf(o.Value)

		if o.Dependencies != nil {
			return fmt.Errorf(
				"fields were specified on object %s when it was provided",
				o,
			)
		}

		if o.Name == "" {
			if !isStructPtr(o.reflectType) {
				return fmt.Errorf(
					"expected unnamed object value to be a pointer to a struct but got type %s "+
						"with value %v",
					o.reflectType,
					o.Value,
				)
			}

			if g.unnamedType == nil {
				g.unnamedType = make(map[reflect.Type]bool)
			}

			if g.unnamedType[o.reflectType] {
				return fmt.Errorf(
					"provided two unnamed instances of type *%s",
					o.reflectType.Elem().String(),
				)
			}
			g.unnamedType[o.reflectType] = true
			g.unnamed = append(g.unnamed, o)
		} else {
			if g.named == nil {
				g.named = make(map[string]*Dew)
			}

			if g.named[o.Name] != nil {
				return fmt.Errorf("provided two instances named %s", o.Name)
			}
			g.named[o.Name] = o
		}

		if g.Logger != nil {
			if o.created {
				g.Logger.Debugf("created %s", o)
			} else {
				g.Logger.Debugf("provided %s", o)
			}
		}
	}
	return nil
}

// Populate the incomplete Objects.
func (g *Graph) Populate() error {
	for _, o := range g.named {
		if o.Complete {
			continue
		}

		if err := g.populateExplicit(o); err != nil {
			return err
		}
	}

	// We append and modify our slice as we go along, so we don't use a standard
	// range loop, and do a single pass thru each object in our graph.
	i := 0
	for {
		if i == len(g.unnamed) {
			break
		}

		o := g.unnamed[i]
		i++

		if o.Complete {
			continue
		}

		if err := g.populateExplicit(o); err != nil {
			return err
		}
	}

	// A Second pass handles injecting Interface values to ensure we have created
	// all concrete types first.
	for _, o := range g.unnamed {
		if o.Complete {
			continue
		}

		if err := g.populateUnnamedInterface(o); err != nil {
			return err
		}
	}

	for _, o := range g.named {
		if o.Complete {
			continue
		}

		if err := g.populateUnnamedInterface(o); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graph) populateExplicit(o *Dew) error {
	// Ignore named value types.
	if o.Name != "" && !isStructPtr(o.reflectType) {
		return nil
	}

StructLoop:
	for i := 0; i < o.reflectValue.Elem().NumField(); i++ {
		field := o.reflectValue.Elem().Field(i)
		fieldType := field.Type()
		fieldName := o.reflectType.Elem().Field(i).Name
		option, ok := o.Options[fieldName]
		if !ok {
			continue
		}

		// Cannot be used with unexported fields.
		if !field.CanSet() {
			return fmt.Errorf(
				"inject requested on unexported field %s in type %s",
				o.reflectType.Elem().Field(i).Name,
				o.reflectType,
			)
		}

		// Don't overwrite existing values.
		if !isNilOrZero(field, fieldType) {
			continue
		}

		// Named injects must have been explicitly provided.
		if option.Name != "" {
			existing := g.named[option.Name]
			if existing == nil {
				return fmt.Errorf(
					"did not find object named %s required by field %s in type %s",
					option.Name,
					o.reflectType.Elem().Field(i).Name,
					o.reflectType,
				)
			}

			if !existing.reflectType.AssignableTo(fieldType) {
				return fmt.Errorf(
					"object named %s of type %s is not assignable to field %s (%s) in type %s",
					option.Name,
					fieldType,
					o.reflectType.Elem().Field(i).Name,
					existing.reflectType,
					o.reflectType,
				)
			}

			field.Set(reflect.ValueOf(existing.Value))
			if g.Logger != nil {
				g.Logger.Debugf(
					"assigned %s to field %s in %s",
					existing,
					o.reflectType.Elem().Field(i).Name,
					o,
				)
			}
			o.addDep(fieldName, existing)
			continue StructLoop
		}

		// Interface injection is handled in a second pass.
		if fieldType.Kind() == reflect.Interface {
			continue
		}

		// Slice are created and required to be private.
		if fieldType.Kind() == reflect.Slice {

			newSlice := reflect.MakeSlice(fieldType, len(option.Vapor), len(option.Vapor))
			for vi, vapor := range option.Vapor {
				existing := g.named[vapor.Dew]
				if existing == nil {
					return fmt.Errorf(
						"did not find object named %s required by field %s in type %s",
						vapor.Dew,
						o.reflectType.Elem().Field(i).Name,
						o.reflectType,
					)
				}

				if !existing.reflectType.AssignableTo(fieldType.Elem()) {
					return fmt.Errorf(
						"object named %s of type %s is not assignable to field %s (%s) in type %s",
						vapor.Dew,
						fieldType.Elem(),
						o.reflectType.Elem().Field(i).Name,
						existing.reflectType,
						o.reflectType,
					)
				}

				newSlice.Index(vi).Set(reflect.ValueOf(existing.Value))
				if g.Logger != nil {
					g.Logger.Debugf(
						"assigned %s to field %s in %s",
						existing,
						o.reflectType.Elem().Field(i).Name,
						o,
					)
				}
				o.addDep(fieldName, existing)
			}
			field.Set(newSlice)
			if g.Logger != nil {
				g.Logger.Debugf(
					"made slice for field %s in %s",
					o.reflectType.Elem().Field(i).Name,
					o,
				)
			}
			continue StructLoop
		}

		// Maps are created and required to be private.
		if fieldType.Kind() == reflect.Map {

			newMap := reflect.MakeMap(fieldType)
			for _, vapor := range option.Vapor {
				valueKey := reflect.New(fieldType.Key()).Elem()
				if err := setFieldWithString(valueKey, vapor.Name); err != nil {
					return err
				}
				existing := g.named[vapor.Dew]
				if existing == nil {
					return fmt.Errorf(
						"did not find object named %s required by field %s in type %s",
						vapor.Dew,
						o.reflectType.Elem().Field(i).Name,
						o.reflectType,
					)
				}

				if !existing.reflectType.AssignableTo(fieldType.Elem()) {
					return fmt.Errorf(
						"object named %s of type %s is not assignable to field %s (%s) in type %s",
						vapor.Dew,
						fieldType.Elem(),
						o.reflectType.Elem().Field(i).Name,
						existing.reflectType,
						o.reflectType,
					)
				}

				newMap.SetMapIndex(valueKey, reflect.ValueOf(existing.Value))
				if g.Logger != nil {
					g.Logger.Debugf(
						"assigned %s to field %s in %s",
						existing,
						o.reflectType.Elem().Field(i).Name,
						o,
					)
				}
				o.addDep(fieldName, existing)
			}
			field.Set(newMap)
			if g.Logger != nil {
				g.Logger.Debugf(
					"made map for field %s in %s",
					o.reflectType.Elem().Field(i).Name,
					o,
				)
			}
			continue StructLoop
		}

		// Can only inject Pointers from here on.
		if !isStructPtr(fieldType) {
			return fmt.Errorf(
				"found inject option on unsupported field %s in type %s",
				o.reflectType.Elem().Field(i).Name,
				o.reflectType,
			)
		}

		// Unless it's a private inject, we'll look for an existing instance of the
		// same type.
		for _, existing := range g.unnamed {
			if existing.reflectType.AssignableTo(fieldType) {
				field.Set(reflect.ValueOf(existing.Value))
				if g.Logger != nil {
					g.Logger.Debugf(
						"assigned existing %s to field %s in %s",
						existing,
						o.reflectType.Elem().Field(i).Name,
						o,
					)
				}
				o.addDep(fieldName, existing)
				continue StructLoop
			}
		}

		newValue := reflect.New(fieldType.Elem())
		newObject := &Dew{
			Value:   newValue.Interface(),
			created: true,
		}

		// Add the newly ceated object to the known set of objects.
		err := g.Provide(newObject)
		if err != nil {
			return err
		}

		// Finally assign the newly created object to our field.
		field.Set(newValue)
		if g.Logger != nil {
			g.Logger.Debugf(
				"assigned newly created %s to field %s in %s",
				newObject,
				o.reflectType.Elem().Field(i).Name,
				o,
			)
		}
		o.addDep(fieldName, newObject)
	}
	return nil
}

func (g *Graph) populateUnnamedInterface(o *Dew) error {
	// Ignore named value types.
	if o.Name != "" && !isStructPtr(o.reflectType) {
		return nil
	}

	for i := 0; i < o.reflectValue.Elem().NumField(); i++ {
		field := o.reflectValue.Elem().Field(i)
		fieldType := field.Type()
		fieldName := o.reflectType.Elem().Field(i).Name
		option, ok := o.Options[fieldName]
		if !ok {
			continue
		}

		// We only handle interface injection here. Other cases including errors
		// are handled in the first pass when we inject pointers.
		if fieldType.Kind() != reflect.Interface {
			continue
		}

		// Don't overwrite existing values.
		if !isNilOrZero(field, fieldType) {
			continue
		}

		// Named injects must have already been handled in populateExplicit.
		if option.Name != "" {
			panic(fmt.Sprintf("unhandled named instance with name %s", option.Name))
		}

		// Find one, and only one assignable value for the field.
		var found *Dew
		for _, existing := range g.unnamed {
			if existing.reflectType.AssignableTo(fieldType) {
				if found != nil {
					return fmt.Errorf(
						"found two assignable values for field %s in type %s. one type "+
							"%s with value %v and another type %s with value %v",
						o.reflectType.Elem().Field(i).Name,
						o.reflectType,
						found.reflectType,
						found.Value,
						existing.reflectType,
						existing.reflectValue,
					)
				}
				found = existing
				field.Set(reflect.ValueOf(existing.Value))
				if g.Logger != nil {
					g.Logger.Debugf(
						"assigned existing %s to interface field %s in %s",
						existing,
						o.reflectType.Elem().Field(i).Name,
						o,
					)
				}
				o.addDep(fieldName, existing)
			}
		}

		// If we didn't find an assignable value, we're missing something.
		if found == nil {
			return fmt.Errorf(
				"found no assignable value for field %s in type %s",
				o.reflectType.Elem().Field(i).Name,
				o.reflectType,
			)
		}
	}
	return nil
}

// Objects returns all known objects, named as well as unnamed. The returned
// elements are not in a stable order.
func (g *Graph) Objects() []*Dew {
	objects := make([]*Dew, 0, len(g.unnamed)+len(g.named))
	for _, o := range g.unnamed {
		objects = append(objects, o)
	}
	for _, o := range g.named {
		objects = append(objects, o)
	}
	// randomize to prevent callers from relying on ordering
	for i := 0; i < len(objects); i++ {
		j := rand.Intn(i + 1)
		objects[i], objects[j] = objects[j], objects[i]
	}
	return objects
}

func (g *Graph) GetDewByName(name string) *Dew {
	if g.named == nil {
		return nil
	}
	return g.named[name]
}

func isStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

func isNilOrZero(v reflect.Value, t reflect.Type) bool {
	switch v.Kind() {
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(t).Interface())
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
}
