package http

const helloPage = `
<!DOCTYPE html>
<html>
<head>
<title>{{.ServerName}}</title>
</head>
<body>
<h2>{{.ServerName}} Echo Service</h2>
<pre>{{.TimeStamp}}</pre>
<h4>REQUEST HEADERS</h4>
{{range $key, $value := .Headers }}
<div><pre>{{$key}} : {{$value}}</pre></div>
{{end}}
<hr/>
<ul>
<li>
You can download the <a href="{{.CertPath}}">{{.ServerName}} Root certificate</a>
</li>
</ul>
</body>
</html>
`
