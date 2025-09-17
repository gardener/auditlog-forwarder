<p>Packages:</p>
<ul>
<li>
<a href="#config.auditlog-forwarder.gardener.cloud%2fv1alpha1">config.auditlog-forwarder.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="config.auditlog-forwarder.gardener.cloud/v1alpha1">config.auditlog-forwarder.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 is a version of the API.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="config.auditlog-forwarder.gardener.cloud/v1alpha1.AuditlogForwarder">AuditlogForwarder
</h3>
<p>
<p>AuditlogForwarder defines the configuration for the audit log forwarder.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>log</code></br>
<em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.Log">
Log
</a>
</em>
</td>
<td>
<p>Log contains the logging configuration for the audit log forwarder.</p>
</td>
</tr>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.Server">
Server
</a>
</em>
</td>
<td>
<p>Server contains the server configuration for the audit log forwarder.</p>
</td>
</tr>
<tr>
<td>
<code>outputs</code></br>
<em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.Output">
[]Output
</a>
</em>
</td>
<td>
<p>Outputs contains the list of outputs to forward audit logs to.</p>
</td>
</tr>
<tr>
<td>
<code>injectAnnotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>InjectAnnotations contains annotations to be injected into audit events.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.auditlog-forwarder.gardener.cloud/v1alpha1.ClientTLS">ClientTLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.OutputHTTP">OutputHTTP</a>)
</p>
<p>
<p>ClientTLS defines the TLS configuration for client.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>caFile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CAFile is the file containing the Certificate Authority to verify the server certificate.</p>
</td>
</tr>
<tr>
<td>
<code>certFile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CertFile is the file containing the client certificate for mutual TLS.</p>
</td>
</tr>
<tr>
<td>
<code>keyFile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>KeyFile is the file containing the client private key for mutual TLS.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.auditlog-forwarder.gardener.cloud/v1alpha1.Log">Log
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.AuditlogForwarder">AuditlogForwarder</a>)
</p>
<p>
<p>Log defines the logging configuration for the audit log forwarder.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>level</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Level is the level/severity for the logs. Must be one of [info,debug,error].</p>
</td>
</tr>
<tr>
<td>
<code>format</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Format is the output format for the logs. Must be one of [text,json].</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.auditlog-forwarder.gardener.cloud/v1alpha1.Output">Output
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.AuditlogForwarder">AuditlogForwarder</a>)
</p>
<p>
<p>Output defines an output to forward audit logs to.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>http</code></br>
<em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.OutputHTTP">
OutputHTTP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTP contains the HTTP output configuration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.auditlog-forwarder.gardener.cloud/v1alpha1.OutputHTTP">OutputHTTP
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.Output">Output</a>)
</p>
<p>
<p>OutputHTTP defines the configuration for an HTTP output.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL is the endpoint URL to send audit logs to.</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.ClientTLS">
ClientTLS
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLS contains the TLS configuration for client.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.auditlog-forwarder.gardener.cloud/v1alpha1.Server">Server
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.AuditlogForwarder">AuditlogForwarder</a>)
</p>
<p>
<p>Server defines the server configuration for the audit log forwarder.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>port</code></br>
<em>
uint
</em>
</td>
<td>
<em>(Optional)</em>
<p>Port is the port that the server will listen on.</p>
</td>
</tr>
<tr>
<td>
<code>address</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Address is the IP address that the server will listen on.
If unspecified all interfaces will be used.</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.TLS">
TLS
</a>
</em>
</td>
<td>
<p>TLS contains the TLS configuration for the server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.auditlog-forwarder.gardener.cloud/v1alpha1.TLS">TLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.auditlog-forwarder.gardener.cloud/v1alpha1.Server">Server</a>)
</p>
<p>
<p>TLS defines the TLS configuration for the server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certFile</code></br>
<em>
string
</em>
</td>
<td>
<p>CertFile is the file containing the x509 Certificate for HTTPS.</p>
</td>
</tr>
<tr>
<td>
<code>keyFile</code></br>
<em>
string
</em>
</td>
<td>
<p>KeyFile is the file containing the x509 private key matching the certificate.</p>
</td>
</tr>
<tr>
<td>
<code>clientCAFile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClientCAFile is the file containing the Certificate Authority to verify client certificates.
If specified, client certificate verification will be enabled with RequireAndVerifyClientCert policy.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
