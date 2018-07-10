package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	//"fmt"
	// "github.com/davecgh/go-spew/spew"
	"regexp"

	"bitbucket.org/dexterchaney/whoville/utils"
	pb "bitbucket.org/dexterchaney/whoville/webapi/rpc/apinator"
	"github.com/graphql-go/graphql"
)

type VaultVals struct {
	ID   string `json: "id"`
	envs []Env  `json: "envs"`
}
type Env struct {
	ID       int       `json: "id"`
	name     string    `json: "name"`
	services []Service `json: "services"`
}
type Service struct {
	envID int    `json: "envID"`
	ID    int    `json: "id"`
	name  string `json: "name"`
	files []File `json: "files"`
}
type File struct {
	envID  int     `json: "envID"`
	servID int     `json: "servID"`
	ID     int     `json: "id"`
	name   string  `json: "name"`
	values []Value `json: "values"`
}
type Value struct {
	envID  int    `json: "envID"`
	servID int    `json: "servID"`
	fileID int    `json: "fileID"`
	ID     int    `json: "id"`
	key    string `json: "name"`
	value  string `json: "value"`
	source string `json: "source"`
}

//GraphQL Accepts a GraphQL query and creates a response
func (s *Server) GraphQL(ctx context.Context, req *pb.GraphQLQuery) (*pb.GraphQLResp, error) {
	rawResult := graphql.Do(graphql.Params{
		Schema:        s.GQLSchema,
		RequestString: req.Query,
	})
	//
	result := &pb.GraphQLResp{}
	resultBytes := bytes.NewBuffer(nil)
	json.NewEncoder(resultBytes).Encode(rawResult)
	json.Unmarshal(resultBytes.Bytes(), result)
	return result, nil
}

//InitGQL Initializes the GQL schema
func (s *Server) InitGQL() {
	//apiClient := pb.NewEnterpriseServiceBrokerProtobufClient("https://localhost:8008", &http.Client{})

	makeVaultReq := &pb.GetValuesReq{}

	// Values Schema
	vault, err := s.GetValues(context.Background(), makeVaultReq)
	utils.LogErrorObject(err, s.Log, true)
	templates, err := s.getTemplateData()
	// (&spew.ConfigState{
	// 	Indent:                  "\t",
	// 	DisablePointerAddresses: true,
	// 	DisableCapacities:       true,
	// 	SortKeys:                true,
	// }).Dump(templates)

	envList := []Env{}
	for i, env := range vault.Envs {
		serviceList := []Service{}
		for j, service := range env.Services {
			fileList := []File{}
			for k, file := range service.Files {
				valList := []Value{}
				l := 0
				for _, val := range file.Values {
					valQL := Value{ID: l, envID: i, servID: j, fileID: k, key: val.Key, value: val.Value, source: "value"}
					valList = append(valList, valQL)
					l++
				}
				// Append template secret keys
				for _, secret := range templates[env.Name].(map[string]interface{})[service.Name].(map[string][]*Value)[file.Name] {
					secret.ID = l
					secret.envID = i
					secret.servID = j
					secret.fileID = k
					valList = append(valList, *secret)
					l++
				}
				fileQL := File{ID: k, envID: i, servID: j, name: file.Name, values: valList}
				fileList = append(fileList, fileQL)
			}
			serviceQL := Service{ID: j, envID: i, name: service.Name, files: fileList}
			serviceList = append(serviceList, serviceQL)
		}
		envQL := Env{ID: i, name: env.Name, services: serviceList}
		envList = append(envList, envQL)
	}
	vaultQL := VaultVals{envs: envList}

	// Convert data to a nested structure
	var ValueObject = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Value",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"envid": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"servid": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"fileid": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"key": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						val := params.Source.(Value).ID
						file := params.Source.(Value).fileID
						serv := params.Source.(Value).servID
						env := params.Source.(Value).envID
						return vaultQL.envs[env].services[serv].files[file].values[val].key, nil
					},
				},
				"value": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {

						val := params.Source.(Value).ID
						file := params.Source.(Value).fileID
						serv := params.Source.(Value).servID
						env := params.Source.(Value).envID
						return vaultQL.envs[env].services[serv].files[file].values[val].value, nil
					},
				},
				"source": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {

						val := params.Source.(Value).ID
						file := params.Source.(Value).fileID
						serv := params.Source.(Value).servID
						env := params.Source.(Value).envID
						return vaultQL.envs[env].services[serv].files[file].values[val].source, nil
					},
				},
			},
		})
	var FileObject = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "File",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"envid": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"servid": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"name": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						file := params.Source.(File).ID
						serv := params.Source.(File).servID
						env := params.Source.(File).envID
						return vaultQL.envs[env].services[serv].files[file].name, nil
					},
				},
				"values": &graphql.Field{
					Type: graphql.NewList(ValueObject),
					Args: graphql.FieldConfigArgument{
						"keyName": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
						"sourceName": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
						// "valName": &graphql.ArgumentConfig{
						// 	Type: graphql.String,
						// },
					},
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						//get list of values and return
						keyStr, keyOK := params.Args["keyName"].(string)
						// valStr, valOK := params.Args["valName"].(string)
						sourceStr, sourceOK := params.Args["sourceName"].(string)

						file := params.Source.(File).ID
						serv := params.Source.(File).servID
						env := params.Source.(File).envID
						values := []Value{}
						if keyOK {
							// Construct a regular expression based on the search
							regex := regexp.MustCompile(`(?i).*` + keyStr + `.*`)
							for i, v := range vaultQL.envs[env].services[serv].files[file].values {
								if regex.MatchString(v.key) {
									values = append(values, vaultQL.envs[env].services[serv].files[file].values[i])
								}
							}
						} else {
							values = vaultQL.envs[env].services[serv].files[file].values
						}

						if sourceOK {
							filteredValues := []Value{}
							for _, value := range values {
								if value.source == sourceStr {
									filteredValues = append(filteredValues, value)
								}
							}
							values = filteredValues
						}
						//else if valOK {
						// 	for i, v := range vaultQL.envs[env].services[serv].files[file].values {
						// 		if v.value == valStr {
						// 			return []Value{vaultQL.envs[env].services[serv].files[file].values[i]}, nil
						// 		}
						// 	}
						// 	return vaultQL.envs[env].services[serv].files[file].values, errors.New("valName not found")
						// }
						return values, nil
					},
				},
			},
		})
	var ServiceObject = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Service",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"envid": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"name": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						serv := params.Source.(Service).ID
						env := params.Source.(Service).envID
						return vaultQL.envs[env].services[serv].name, nil
					},
				},
				"files": &graphql.Field{
					Type: graphql.NewList(FileObject),
					Args: graphql.FieldConfigArgument{
						"fileName": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
					},
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						fileStr, isOK := params.Args["fileName"].(string)
						serv := params.Source.(Service).ID
						env := params.Source.(Service).envID
						if isOK {
							for i, f := range vaultQL.envs[env].services[serv].files {
								if f.name == fileStr {
									return []File{vaultQL.envs[env].services[serv].files[i]}, nil
								}
							}
							return vaultQL.envs[env].services[serv].files, errors.New("fileName not found")
						}
						return vaultQL.envs[env].services[serv].files, nil
					},
				},
			},
		})
	var EnvObject = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Env",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"name": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						env := params.Source.(Env).ID
						return vaultQL.envs[env].name, nil
					},
				},
				"services": &graphql.Field{
					Type: graphql.NewList(ServiceObject),
					Args: graphql.FieldConfigArgument{
						"servName": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
					},

					Resolve: func(params graphql.ResolveParams) (interface{}, error) {

						servStr, isOK := params.Args["servName"].(string)
						env := params.Source.(Env).ID
						if isOK {
							for i, s := range vaultQL.envs[env].services {
								if s.name == servStr {
									return []Service{vaultQL.envs[env].services[i]}, nil
								}
							}
							return vaultQL.envs[env].services, errors.New("servName not found")
						}
						return vaultQL.envs[env].services, nil

					},
				},
			},
		})
	var VaultValObject = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "VaultVals",

			Fields: graphql.Fields{
				"envs": &graphql.Field{
					Type: graphql.NewList(EnvObject),
					Args: graphql.FieldConfigArgument{
						"envName": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
					},
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						if envStr, isOK := params.Args["envName"].(string); isOK {
							for i, e := range vaultQL.envs {
								if e.name == envStr {
									return []Env{vaultQL.envs[i]}, nil
								}
							}
							return vaultQL.envs, errors.New("envName not found")
						}
						return vaultQL.envs, nil

					},
				},
			},
		})
	s.GQLSchema, _ = graphql.NewSchema(graphql.SchemaConfig{
		Query: VaultValObject,
	})

	// fmt.Println("Test with: curl -g 'http://localhost:8008/graphql?query={envs{services{files{values{key,value}}}}}'")
}
