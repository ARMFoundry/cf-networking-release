# 3rd Party Plugin Development for Container Networking

*If you want to integrate your own CNI plugin with Cloud Foundry, read this document.*

If you have any questions or feedback, please visit the `#container-networking` channel on [Cloud Foundry Slack](http://slack.cloudfoundry.org/).

## MTU
CNI plugins should automatically detect the MTU settings on the host, and set the MTU
on container network interfaces appropriately.  For example, if the host MTU is 1500 bytes
and the plugin encapsulates with 50 bytes of header, the plugin should ensure that the
container MTU is no greater than 1450 bytes.  This is to ensure there is no fragmentation.
The built-in flannel CNI plugin does this.

## To replace flannel with your own CNI plugin
0. Remove the following BOSH jobs:
  - `cni-flannel`
  - `vxlan-policy-agent`
0. Remove the following BOSH packages:
  - `flannel`
  - `flannel-watchdog`
0. Add in all packages and jobs required by your CNI plugin.  At a minimum, you must provide a CNI binary program and a CNI config file.
  - For more info on **bosh packaging scripts** read [this](http://bosh.io/docs/packages.html#create-a-packaging-script).
  - For more info on **bosh jobs** read [this](http://bosh.io/docs/jobs.html).
0. Update the [deployment manifest properties](http://bosh.io/docs/deployment-manifest.html#properties)

  ```yaml
  properties:
    cf_networking:
      garden_external_networker:
        cni_plugin_dir: /var/vcap/packages/YOUR_PACKAGE/bin # directory for CNI binaries
        cni_config_dir: /var/vcap/jobs/YOUR_JOB/config/cni  # directory for CNI config file(s)
  ```
  Remove any lingering references to `vxlan-policy-agent` in the deployment manifest, and replace the `plugin` properties
  with any manifest properties that your bosh job requires.

## What data will my CNI plugin receive?
The `garden-external-networker` will invoke one or more CNI plugins, according to the [CNI Spec](https://github.com/containernetworking/cni/blob/master/SPEC.md).
It will start with the CNI config files available in the `cni_config_dir` and also inject
some dynamic information about the container, including the CloudFoundry App, Space and Org that it belongs to.

For example, in the included networking stack, we have a `wrapper` CNI plugin.
At deploy time, its config is generated from this [template](../jobs/cni-flannel/templates/30-cni-wrapper-plugin.conf.erb),
but when the container is being created, the CNI plugin receives data like this:

```json
{
  {
    "name": "cni-wrapper",
    "type": "cni-wrapper-plugin",
    "cniVersion": "0.2.0",
    "datastore": "/var/vcap/data/container-metadata/store.json",
    "iptables_lock_file": "/var/vcap/data/garden-cni/iptables.lock",
    "overlay_network": "10.255.0.0/16",
    "delegate": {
      "name": "cni-flannel",
      "type": "flannel",
      "subnetFile": "/var/vcap/data/flannel/subnet.env",
      "dataDir": "/var/vcap/data/flannel/data",
      "delegate": {
        "bridge": "cni-flannel0",
        "isDefaultGateway": true,
        "ipMasq": false
       }
    }
  },
  "metadata": {
    "app_id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
    "org_id": "2ac41bbf-8eae-4f28-abab-51ca38dea3e4",
    "policy_group_id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
    "space_id": "4246c57d-aefc-49cc-afe0-5f734e2656e8"
  }
}
```

## To deploy a local-only (no-op) CNI plugin
As a baseline, you can deploy using only the basic [bridge CNI plugin](https://github.com/containernetworking/cni/blob/master/Documentation/bridge.md).

This plugin will provide connectivity between containers on the same Garden host (Diego cell)
but will not provide a cross-host network.  However, it can be a useful baseline configuration for
testing and development.

```bash
cd bosh-lite
bosh target lite
bosh update cloud-config cloud-config.yml
bosh deployment local-only.yml
bosh deploy
```



## Policy Server Internal API
If you are replacing the built-in "VXLAN Policy Agent" with your own Policy Enforcement implementation,
you can use the Policy Server's internal API to retrieve policy information.

There is a single endpoint to retrieve policies:

`GET https://policy-server.service.cf.internal:4003/networking/v0/internal/policies`

Additionally, you can use the `id` query parameter to filter the response to include
only policies with a source or destination that match any of the comma-separated
`group_policy_id`'s that are included.

### TLS configuration
The Policy Server internal API requires Mutual TLS.  All connections must use a client certificate
that is signed by a trusted certificate authority.  The certs and keys should be configured via BOSH manifest
properties on the Policy Server and on your custom policy client, e.g.

```yaml
properties:
  cf_networking:
    policy_server:
      ca_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_CA_CERT
        -----END CERTIFICATE-----
      server_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_SERVER_CERT
        -----END CERTIFICATE-----
      server_key: |
        -----BEGIN RSA PRIVATE KEY-----
        REPLACE_WITH_SERVER_KEY
        -----END RSA PRIVATE KEY-----

  your_networking_provider:
    your_policy_client:
      ca_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_CA_CERT
        -----END CERTIFICATE-----
      client_cert: |
        -----BEGIN CERTIFICATE-----
        REPLACE_WITH_CLIENT_CERT
        -----END CERTIFICATE-----
      client_key: |
        -----BEGIN RSA PRIVATE KEY-----
        REPLACE_WITH_CLIENT_KEY
        -----END RSA PRIVATE KEY-----
```

The server requires that connections use the TLS cipher suite
`TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`.  Your client must support this cipher suite.

We provide [a script](../scripts/generate-certs) to generate all required certs & keys.

### Policy Server Internal API Details

`GET /networking/v0/internal/policies`

List all policies optionally filtered to match requested  `policy_group_id`'s

Query Parameters (optional):

- `id`: comma-separated `policy_group_id` values

Response Body:

- `policies`: list of policies
- `policies[].destination`: the destination of the policy
- `policies[].destination.id`: the `policy_group_id` of the destination (currently always an `app_id`)
- `policies[].destination.port`: the `port` allowed on the destination
- `policies[].destination.protocol`: the `protocol` allowed on the destination: `tcp` or `udp`
- `policies[].destination.tag`: the `tag` of the source allowed to the destination
- `policies[].source`: the source of the policy
- `policies[].source.id`: the `policy_group_id` of the source (currently always an `app_id`)
- `policies[].source.tag`: the `tag` of the source allowed to the destination

### Examples Requests and Responses

#### Get all policies

```bash
curl -s \
  --cacert certs/ca.crt \
  --cert certs/client.crt \
  --key certs/client.key \
  https://policy-server.service.cf.internal:4003/networking/v0/internal/policies
```

```json
  {
      "policies": [
        {
            "destination": {
                "id": "eb95ff20-cba8-4edc-8f4a-cf80d0669faf",
                "port": 8080,
                "protocol": "tcp",
                "tag": "0002"
            },
            "source": {
                "id": "4a2d3627-0b8c-42d1-9563-22696eedc05d",
                "tag": "0001"
            }
        },
        {
            "destination": {
                "id": "b611f7e6-c8fe-41cb-b150-92581aafa5c2",
                "port": 8080,
                "protocol": "tcp",
                "tag": "0004"
            },
            "source": {
                "id": "3b348978-a3cb-487c-a277-58fdc3e2c678",
                "tag": "0003"
            }
        },
        {
            "destination": {
                "id": "8fa287c9-0d01-491e-a1d5-d6e2d8a1ef61",
                "port": 8080,
                "protocol": "tcp",
                "tag": "0005"
            },
            "source": {
                "id": "8fa287c9-0d01-491e-a1d5-d6e2d8a1ef61",
                "tag": "0005"
            }
        },
        {
            "destination": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "port": 5555,
                "protocol": "tcp",
                "tag": "0006"
            },
            "source": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "tag": "0006"
            }
        }
    ]
}
```

#### Get filtered policies

Returns all policies with source or destination id's that match any of the
included `policy_group_id`'s.

```bash
curl -s \
--cacert certs/ca.crt \
--cert certs/client.crt \
--key certs/client.key \
https://policy-server.service.cf.internal:4003/networking/v0/internal/policies?id=5351a742-6704-46df-8de0-1a376adab65c,d5bbc5ed-886a-44e6-945d-67df1013fa16
```

```json
{
    "policies": [
        {
            "destination": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "port": 5555,
                "protocol": "tcp",
                "tag": "0006"
            },
            "source": {
                "id": "d5bbc5ed-886a-44e6-945d-67df1013fa16",
                "tag": "0006"
            }
        },
        {
            "destination": {
                "id": "5351a742-6704-46df-8de0-1a376adab65c",
                "port": 5555,
                "protocol": "tcp",
                "tag": "0007"
            },
            "source": {
                "id": "5351a742-6704-46df-8de0-1a376adab65c",
                "tag": "0007"
            }
        }
    ]
}
```
