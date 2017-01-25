## Deploy to bosh-lite

Follow the instructions [here](https://github.com/cloudfoundry/bosh-lite) to install `bosh-lite` on your machine.

Ensure that `br_netfilter` is enabled on your vagrant box:
```bash
pushd ~/workspace/bosh-lite
  vagrant ssh -c 'sudo modprobe br_netfilter'
popd
```
or edit your `Vagrantfile` to include
```ruby
config.vm.provision "shell", inline: "sudo modprobe br_netfilter"
```

Upload the latest `bosh-lite` stemcell 
```bash
bosh upload stemcell https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
```

Or download the stemcell and manually upload it to `bosh-lite` (potentially faster)
```bash
curl -L -o bosh-lite-stemcell-latest.tgz https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
bosh upload stemcell bosh-lite-stemcell-latest.tgz
```

Then grab the required releases
```bash
pushd ~/workspace
  git clone https://github.com/cloudfoundry/diego-release
  git clone https://github.com/cloudfoundry/cf-release
  git clone https://github.com/cloudfoundry-incubator/cf-networking-release
popd
```

Deploy:
- Option 1: use the script
  ```bash
  pushd ~/workspace/cf-networking-release
    ./scripts/deploy-to-bosh-lite
  popd
  ```

- Option 2: deploy by hand
  This assumes you're comfortable with BOSH.  First acquire `cf-release`, `diego-release` and [all of its dependencies](https://github.com/cloudfoundry/diego-release/tree/develop/examples/bosh-lite).  Upload to your bosh director.
  At a minimum, you'll need to do something like
  ```
  bosh upload release https://bosh.io/d/github.com/cloudfoundry/garden-runc-release
  bosh upload release https://bosh.io/d/github.com/cloudfoundry/cflinuxfs2-rootfs-release
  bosh upload release https://bosh.io/d/github.com/cloudfoundry-incubator/etcd-release
  ```

  Then
  ```bash
  pushd ~/workspace/cf-networking-release
    bosh upload release releases/cf-networking-<LATEST-VERSION>.yml

    ./scripts/generate-bosh-lite-manifests
    bosh -d bosh-lite/deployments/cf_networking.yml deploy
    bosh -d bosh-lite/deployments/diego_cf_networking.yml deploy
  popd
  ```

## Kicking the tires

Try out our [Cats and Dogs example](../src/example-apps/cats-and-dogs) on your new deployment.


## Known issues with bosh-lite
There is a known issue where VMs on `bosh-lite` can start failing,
particularly if the host machine goes to sleep.

If you run `bosh vms` and see any failing VMs, then you can either recreate the
individual failing vm(s) with
```
bosh recreate <vm_name>
```
or you can run
```
bosh deploy --recreate
```
to recreate all VMs.

## Syslog forwarding
This is not specific to CF Networking, but is useful for debugging during development.

To forward all logs from your bosh-lite to a syslog destination (like Papertrail),
add the following block to `manifest-generation/stubs/bosh-lite-cf.yml`:
```yaml
  syslog_daemon_config:
    address: some-syslog-host.example.com
    port: 12345
    transport: udp
```
