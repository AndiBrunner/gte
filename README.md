# GTE
Inspired by the dockerize template library, GTE is a go template engine based on the golang template package and the go-jmespath library (JMESPath is a query language for JSON). 

### Usage:
gte [options] template destination [template destination]...  

### Options:
	-n --no-overwrite
        Do not overwrite destination file if it already exists.
	-d --delims
        template tag delimiters. default "{{":"}}"
### Arguments:
 Arguments:
 
   template     - the template. Can be passed multiple times together with destination as an ordered pair (2-tuple). Does also support directories. <br>
   destination  - the destination . Can be passed multiple times together with destination as an ordered pair (2-tuple). Does also support directories.

### Examples:
  
Generate test.conf using test.tmpl as a template.
   
   gte test.tmpl /etc/test/test.conf

<br>

 content template test.tmpl:
 ```
 #Begin Config
 
 json:
 {{ jsonQuery `{"cc":{"servers":[{"host":"aa","port":1001},{"host":"bb","port":1002},{"host":"cc","port":1003}]}}` `-i2 cc.servers[*]` }}
 
#End Config
``` 
generated test.conf:
```
#Begin Config

json:
[
  {
    "host": "aa",
    "port": 1001
  },
  {
    "host": "bb",
    "port": 1002
  },
  {
    "host": "cc",
    "port": 1003
  }
] 

#End Config
```

### Syntax in Go Template:
```
{{ jsonQuery \`json-source\` \`[options] jmespath\` }} 
```

#### json-source: json 
  e.g. \`{"host": "10.0.0.1"}`
<br>
  e.g. .Env.VCAP_SERVICES (json in environment variable)
  
#### jmespath: valid jmespath (see http://jmespath.org)
  e.g. \`host\`
<br>
  e.g. \`services[0].host\`
<br>
  e.g. \`services[?host=='10.0.0.1']\`
  
#### options:
```    
(none)
output rendered as inline json
```

```    
-iNUM
intend json output with the defined number of spaces, e.g. -i5
```    

```    
-y
yaml output
```    
