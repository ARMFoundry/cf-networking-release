require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'policy-server job template rendering' do
    let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
    let(:release) {ReleaseDir.new(release_path)}
    let(:job) {release.job('policy-server')}

    let(:merged_manifest_properties) do
      {
        'disable' => false,
        'policy_cleanup_interval' => 1,
        'max_policies_per_app_source' => 2,
        'enable_space_developer_self_service' => true,
        'listen_ip' => '111.11.11.1',
        'listen_port' => 1234,
        'debug_port' => 2345,
        'uaa_client' => 'some-uaa-client',
        'uaa_client_secret' => 'some-uaa-client-secret',
        'uaa_ca' => 'some-uaa-ca',
        'uaa_hostname' => 'some-uaa-hostname',
        'uaa_port' => 3456,
        'cc_hostname' => 'some-cc-hostname',
        'cc_port' => 4567,
        'skip_ssl_validation' => true,
        'database' => {
          'type' => 'postgres',
          'host' => 'some-database-host',
          'username' => 'some-database-username',
          'password' => 'some-database-password',
          'port' => 5678,
          'name' => 'some-database-name',
          'require_ssl' => true,
          'ca_cert' => 'some-ca-cert',
          'connect_timeout_seconds' => 3,
        },
        'max_idle_connections' => 4,
        'max_open_connections' => 5,
        'tag_length' => 4,
        'metron_port' => 6789,
        'log_level' => 'debug',
        'allowed_cors_domains' => ['some-cors-domain'],
      }
    end

    describe 'database_ca.crt' do
      let(:template) {job.template('config/certs/database_ca.crt')}
      it 'writes the content of database.ca_cert' do
        merged_manifest_properties['database']['ca_cert'] = 'the ca cert'
        expect(template.render(merged_manifest_properties)).to eq('the ca cert')
      end
    end

    describe 'policy-server.json' do
      let(:template) {job.template('config/policy-server.json')}

      it 'creates a config/policy-server.json from properties' do
        config = JSON.parse(template.render(merged_manifest_properties))
        expect(config).to eq({
          'listen_host' => '111.11.11.1',
          'listen_port' => 1234,
          'log_prefix' => 'cfnetworking',
          'debug_server_host' => '127.0.0.1',
          'debug_server_port' => 2345,
          'uaa_client' => 'some-uaa-client',
          'uaa_client_secret' => 'some-uaa-client-secret',
          'uaa_url' => 'https://some-uaa-hostname',
          'uaa_port' => 3456,
          'cc_url' => 'http://some-cc-hostname:4567',
          'skip_ssl_validation' => true,
          'database' => {
            'type' => 'postgres',
            'user' => 'some-database-username',
            'password' => 'some-database-password',
            'host' => 'some-database-host',
            'port' => 5678,
            'timeout' => 3,
            'database_name' => 'some-database-name',
            'require_ssl' => true,
            'ca_cert' => '/var/vcap/jobs/policy-server/config/certs/database_ca.crt'
          },
          'max_idle_connections' => 4,
          'max_open_connections' => 5,
          'tag_length' => 4,
          'metron_address' => '127.0.0.1:6789',
          'log_level' => 'debug',
          'cleanup_interval' => 60,
          'max_policies' => 2,
          'enable_space_developer_self_service' => true,
          'allowed_cors_domains' => ['some-cors-domain'],
          'uaa_ca' => '/var/vcap/jobs/policy-server/config/certs/uaa_ca.crt',
          'request_timeout' => 5,
        })
      end

      it 'raises an error when the driver (type) is unknown' do
        merged_manifest_properties['database']['type'] = 'bar'
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('unknown driver bar')
      end

      it 'raises an error when the driver (type) is missing' do
        merged_manifest_properties['database'].delete('type')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.type must be specified')
      end

      it 'raises an error when missing username' do
        merged_manifest_properties['database'].delete('username')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.username must be specified')
      end

      it 'raises an error when missing password' do
        merged_manifest_properties['database'].delete('password')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.password must be specified')
      end

      it 'raises an error when missing port' do
        merged_manifest_properties['database'].delete('port')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.port must be specified')
      end

      it 'raises an error when missing name' do
        merged_manifest_properties['database'].delete('name')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.name must be specified')
      end

      it 'raises an error when the cleanup interval is too short' do
        merged_manifest_properties['policy_cleanup_interval'] = 0.7
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('policy_cleanup_interval must be at least 1 minute')
      end
    end
  end
end
