<table border="1" class="nice">
  <tr>
    <th style="width:35%">Container-to-Container Networking Stub Contents</th>
    <th>Editing Instructions</th>
  </tr>
  <tr>
    <td><pre><code>
properties:
  cf_networking:
    vxlan_policy_agent:
      policy_server_url: [...]
      ca_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE-WITH-CA-CERTIFICATE
        -----END CERTIFICATE-----
      client_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE-WITH-CLIENT-CERTIFICATE
        -----END CERTIFICATE-----
      client_key: |
        -----BEGIN EXAMPLE RSA PRIVATE KEY-----
        REPLACE-WITH-CLIENT-KEY
        -----END EXAMPLE RSA PRIVATE KEY-----</code></pre></td>
    <td>Copy in certificates and keys for the policy agent.
        The policy agent communicates with the policy server through TLS.
        See the <a href="../../concepts/understand-cf-networking.html#architecture">Architecture</a> section for more information.
    </td>
  <tr>
    <td><pre><code>
policy_server:
  debug_server_port: REPLACE-WITH-LISTEN-PORT

vxlan_policy_agent:
  debug_server_port: REPLACE-WITH-LISTEN-PORT
</code></pre></td>
    <td>By default, the Policy Server and VXLAN Policy Agent listen on port 22222.<br><br>
    (Optional) Change these port numbers by adding the <code>debug_server_port</code> key pair to the stub file.
    <br>
    Replace <code>REPLACE-WITH-LISTEN-PORT</code> with a port number.<br><br>
    For more information, see <a href="#debug-logging">Manage Debug Logging</a> below.
    </td>
    <tr>
    <td><pre><code>
vxlan_policy_agent:
  iptables_c2c_logging: true
</code></pre></td>
    <td>The default value for <code>iptables_c2c_logging</code> is <code>false</code>.
    <br><br>
    (Optional) Change the value to <code>true</code> to enable logging for Container-to-Container policy iptables rules.
    </td>
  <tr>
    <td><pre><code>
garden_external_networker:
  iptables_asg_logging: true
    </code></pre></td>
    <td>
    The default value for <code>iptables_asg_logging</code> is <code>false</code>.
    <br><br>
    (Optional) Change the value to <code>true</code> to enable
    logging for Application Security Group (ASG) iptables rules.
    </td>
  </tr>
  <tr>
    <td><pre><code>
policy_server:
  uaa_client_secret: REPLACE-WITH-UAA-CLIENT-SECRET
    </code></pre></td>
    <td>Copy in the <code>REPLACE-WITH-UAA-CLIENT-SECRET</code> value you used in the step <a href="#uaa-secret">above</a>.
    </td>
  </tr>
  <tr>
    <td><pre><code>
database:
  type: REPLACE-WITH-DB-TYPE
  username: REPLACE-WITH-USERNAME
  password: REPLACE-WITH-PASSWORD
  host: REPLACE-WITH-DB-HOSTNAME
  port: REPLACE-WITH-DB-PORT
  name: REPLACE-WITH-DB-NAME
   </code></pre></td>
    <td>
    Supply the details for the database from <a href="#enable">step 1</a>.<br>
    The database type must be <code>postgres</code> or <code>mysql</code>.<br>
    Choose a username and password.
    <br>For <code>host</code>, enter the IP address of the database instance.
    <br>Supply a port. For MySQL, a typical port is <code>3360</code>.
    <br>Supply the name of the database.
    </td>
  </tr>
  <tr>
    <td><pre><code>
  ca_cert: |
    -----BEGIN CERTIFICATE-----
    REPLACE-WITH-CA-CERTIFICATE
    -----END CERTIFICATE-----
  server_cert: |
    -----BEGIN CERTIFICATE-----
    REPLACE-WITH-SERVER-CERT
    -----END CERTIFICATE-----
  server_key: |
    -----BEGIN EXAMPLE RSA PRIVATE KEY-----
    REPLACE-WITH-SERVER-KEY
    -----END EXAMPLE RSA PRIVATE KEY-----
garden_external_networker:
  cni_plugin_dir: /var/vcap/packages/flannel/bin
  cni_config_dir: /var/vcap/jobs/cni-flannel/config/cni
    </code></pre></td>
    <td>
    Copy in the certificates and keys for the policy server. 
    The policy server communicates with the policy agent through TLS. 
    See the <a href="../../concepts/understand-cf-networking.html#architecture">Architecture</a> section for more information.
    </td>
  </tr>
  <tr>
    <td><pre><code>
properties:
  cf_networking:
    network: REPLACE-WITH-OVERLAY-NETWORK-CIDR
    </code></pre></td>
    <td>(Optional) Enter an IP range for the overlay network. The CIDR must specify an RFC 1918 range. If you do not set a custom range, the deployment uses <code>10.255.0.0/16</code>.
<br><br>See <a href="../../concepts/understand-cf-networking.html#app-comm">App Instance Communication</a> for more information.
    </td>
  </tr>
  <tr>
    <td><pre><code>
properties:
  cf_networking:
    mtu: REPLACE-WITH-MTU
    </code></pre></code>
    <td>(Optional) You can manually configure the Maximum Transmission Unit (MTU) value to support additional encapsulation overhead.
    </td>
  </tr>
</table>
<li>
Create a file that contains the following bash script. Name the file <code>generate_diego.sh</code>.
<pre>
set -e -x -u

environment_path=STUBS-DIRECTORY
output_path=MANIFEST-DIRECTORY
diego_release_path=LOCAL-DIEGO-REPOSIORY

pushd cf-release
  ./scripts/generate_deployment_manifest aws \
    ${environment_path}/stubs/director-uuid.yml \
    ${diego_release_path}/examples/aws/stubs/cf/diego.yml \
    ${environment_path}/stubs/cf/properties.yml \
    ${environment_path}/stubs/cf/instance-count-overrides.yml \
    ${environment_path}/stubs/cf/stub.yml \
    > ${output_path}/cf.yml
popd

pushd diego-release
  ./scripts/generate-deployment-manifest \
    -g \
    -c ${output_path}/cf.yml \
    -i ${environment_path}/stubs/diego/iaas-settings.yml \
    -p ${environment_path}/stubs/diego/property-overrides.yml \
    -n ${environment_path}/stubs/diego/instance-count-overrides.yml \
    -N ${environment_path}/stubs/cf-networking/stub.yml \
    -v ${environment_path}/stubs/diego/release-versions.yml \
    > ${output_path}/diego.yml
popd
</pre>
Replace the variables as follows:<br>
<ul>
<li><code>STUBS-DIRECTORY</code>: The directory containing your stubs for CF, Diego, and Container-to-Container Networking.</li>
<li><code>MANIFEST-DIRECTORY</code>: The directory where you want the manifest created.</li>
<li><code>LOCAL-DIEGO-REPOSIORY</code>: The directory of the local copy of the <code>diego-release</code> repository.</li>
</ul>
</li>
<li>
Enter the following commands to make the script executable and run the script.
<pre class="terminal">
$ chmod u+x generate_diego.sh
$ ./generate_diego.sh
</pre>
</li>
<li>
Enter the following command to target your BOSH director:
<pre class="terminal">
$ bosh -e MY-ENV alias-env BOSH-DIRECTOR-IP
</pre>
For example,
<pre class="terminal">$ bosh -e stubs alias-env 192.0.2.1</pre>
</li>
<li>
Enter the following command to deploy CF.
<pre class="terminal">$ bosh deploy</pre>
</li>
<li>
Enter the following command to deploy Diego.
<pre class="terminal">$ bosh deploy</pre>
</li>
<li>(Optional) Try the <a href="https://github.com/cloudfoundry-incubator/cf-networking-release/tree/develop/src/example-apps/cats-and-dogs">Cats and Dogs</a> example in the Container-to-Container Networking Release repository. In this tutorial, you deploy two apps and create a Container-to-Container Networking policy that allows them to communicate directly with each other. 
 </li>
</ol>