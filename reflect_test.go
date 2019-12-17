package jsonschema

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type GrandfatherType struct {
	FamilyName string `json:"family_name" jsonschema:"required"`
}

type SomeBaseType struct {
	SomeBaseProperty     int `json:"some_base_property" jsonschema:"omitempty"`
	SomeBasePropertyYaml int `yaml:"some_base_property_yaml"  jsonschema:"omitempty"`
	// The jsonschema required tag is nonsensical for private and ignored properties.
	// Their presence here tests that the fields *will not* be required in the output
	// schema, even if they are tagged required.
	somePrivateBaseProperty   string          `json:"i_am_private" jsonschema:"required"`
	SomeIgnoredBaseProperty   string          `json:"-" jsonschema:"required"`
	SomeSchemaIgnoredProperty string          `jsonschema:"-,required"`
	Grandfather               GrandfatherType `json:"grand"  jsonschema:"omitempty"`

	SomeUntaggedBaseProperty           bool `jsonschema:"required"`
	someUnexportedUntaggedBaseProperty bool
}

type MapType map[string]interface{}

type nonExported struct {
	PublicNonExported  int `jsonschema:"omitempty"`
	privateNonExported int
}

type ProtoEnum int32

func (ProtoEnum) EnumDescriptor() ([]byte, []int) { return []byte(nil), []int{0} }

const (
	Unset ProtoEnum = iota
	Great
)

type TestUser struct {
	SomeBaseType
	nonExported
	MapType

	ID      int                    `json:"id" jsonschema:"required"`
	Name    string                 `json:"name" jsonschema:"required,minLength=1,maxLength=20,pattern=.*,description=this is a property,title=the name,example=joe,example=lucy,default=alex"`
	Friends []int                  `json:"friends,omitempty" jsonschema:"omitempty" jsonschema_description:"list of IDs, omitted when empty"`
	Tags    map[string]interface{} `json:"tags,omitempty"  jsonschema:"omitempty"`

	TestFlag       bool `jsonschema:"omitempty"`
	IgnoredCounter int  `json:"-"`

	// Tests for RFC draft-wright-json-schema-validation-00, section 7.3
	BirthDate time.Time `json:"birth_date,omitempty" jsonschema:"omitempty"`
	Website   url.URL   `json:"website,omitempty" jsonschema:"omitempty"`
	IPAddress net.IP    `json:"network_address,omitempty" jsonschema:"omitempty"`

	// Tests for RFC draft-wright-json-schema-hyperschema-00, section 4
	Photo []byte `json:"photo,omitempty" jsonschema:"required"`

	// Tests for jsonpb enum support
	Feeling ProtoEnum `json:"feeling,omitempty" jsonschema:"omitempty"`
	Age     int       `json:"age" jsonschema:"omitempty,minimum=18,maximum=120,exclusiveMaximum=true,exclusiveMinimum=true"`
	Email   string    `json:"email" jsonschema:"omitempty,format=email"`
}

type CustomTime time.Time

type CustomTypeField struct {
	CreatedAt CustomTime
}

type TestEnum struct {
	Name      string      `json:"name" jsonschema:"required,enum=john,enum=landy,enum=king,enum=john"`
	Age       int         `json:"age" jsonschema:"omitempty,enum=10,enum=20,enum=30,enum=30"`
	Hello     interface{} `json:"hello" jsonschema_enum:"[\"a\",\"b\",\"b\",2,2,null]"`
	EmptyTest string      `json:"emptyTest" jsonschema:"emum="`
}

func TestSchemaGeneration(t *testing.T) {
	tests := []struct {
		typ       interface{}
		reflector *Reflector
		fixture   string
	}{
		{&TestUser{}, &Reflector{}, "fixtures/defaults.json"},
		{&TestUser{}, &Reflector{AllowAdditionalProperties: true}, "fixtures/allow_additional_props.json"},
		{&TestUser{}, &Reflector{RequiredFromJSONSchemaTags: true}, "fixtures/required_from_jsontags.json"},
		{&TestUser{}, &Reflector{ExpandedStruct: true}, "fixtures/defaults_expanded_toplevel.json"},
		{&TestUser{}, &Reflector{IgnoredTypes: []interface{}{GrandfatherType{}}}, "fixtures/ignore_type.json"},
		{&CustomTypeField{}, &Reflector{
			TypeMapper: func(i reflect.Type) *Type {
				if i == reflect.TypeOf(CustomTime{}) {
					return &Type{
						Type:   "string",
						Format: "date-time",
					}
				}
				return nil
			},
		}, "fixtures/custom_type.json"},
		{&TestEnum{}, &Reflector{RequiredFromJSONSchemaTags: true}, "fixtures/enum.json"},
	}

	for _, tt := range tests {
		name := strings.TrimSuffix(filepath.Base(tt.fixture), ".json")
		t.Run(name, func(t *testing.T) {
			f, err := ioutil.ReadFile(tt.fixture)
			require.NoError(t, err)

			actualSchema := tt.reflector.Reflect(tt.typ)
			expectedSchema := &Schema{}

			err = json.Unmarshal(f, expectedSchema)
			require.NoError(t, err)

			expectedJSON, _ := json.MarshalIndent(expectedSchema, "", "  ")
			actualJSON, _ := json.MarshalIndent(actualSchema, "", "  ")
			require.Equal(t, string(expectedJSON), string(actualJSON))
		})
	}
}
