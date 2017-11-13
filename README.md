# gte
Inspired by the dockerize template library, GTE is a go template engine based on the golang template package and the go-jmespath library (JMESPath is a query language for JSON). 

### Usage:
gte [options] template:dest

### Options:
	-n --no-overwrite
        Do not overwrite destination file if it already exists.
	-d --delims
        template tag delimiters. default "{{":"}}"
### Arguments:
  template:dest - Template (/template:/dest). Can be passed multiple times. Does also support directories.

### Examples:
  
   Generate test.conf using test.tmpl as a template.
   
   gte test.tmpl:/etc/test/test.conf
   
   content template test.tmpl:
     1 #Begin Config
     2
     3 json:
     4 {{ jsonQuery `{"cc":{"servers":[{"host":"aa","port":1001},{"host":"bb","port":1002},{"host":"cc","port":1003}]}}` `-i2 cc.servers[*]` }}
     5
     6 #End Config
     
   generated test.conf:
     1 #Begin Config
     2
     3 json:
     4 [
     5   {
     6     "host": "aa",
     7     "port": 1001
     8   },
     9   {
    10     "host": "bb",
    11     "port": 1002
    12   },
    13   {
    14     "host": "cc",
    15     "port": 1003
    16   }
    17 ] 
    18
    19 #End Config

### Syntax in Go Template:
{{ jsonQuery `json-source` `[options] jmespath` }} 

#### json-source: json 
  e.g. `{"host": "10.0.0.1"}`
  e.g. .Env.VCAP_SERVICES (json in environment variable)
  
#### jmespath: valid jmespath (see http://jmespath.org)
  e.g. `host`
  e.g. `services[0].host`
  
#### options:
  (none)
      json inline output
  -iNUM 
      intend json output with the defined number of spaces, e.g. -i5
  -y
      yaml output
