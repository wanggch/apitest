module apitest

go 1.21

require (
	github.com/tidwall/gjson v1.0.0
	gopkg.in/yaml.v3 v3.0.0
)

replace github.com/tidwall/gjson => ./third_party/github.com/tidwall/gjson

replace gopkg.in/yaml.v3 => ./third_party/gopkg.in/yaml.v3
