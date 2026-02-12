import React, {ChangeEvent} from 'react';
import {Button, InlineField, Input, SecretInput} from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const onUrlChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      url: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  const onTokenChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        token: event.target.value,
      },
    });
  };

  const onUsernameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      username: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  const onResetToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        token: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        token: '',
      },
    });
  };

  const { jsonData, secureJsonFields } = options;
  const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;

  return (
    <div className="gf-form-group">
      <InlineField label="URL" labelWidth={12} htmlFor="config-url"
                   tooltip="The URL is the root URL for your Atlassian instance (example: https://achan.atlassian.net)">
        <Input
          id="config-url"
          onChange={onUrlChange}
          value={jsonData.url || ''}
          placeholder="url for your jira instance"
          width={40}
        />
      </InlineField>
      <InlineField label="Email" labelWidth={12} htmlFor="config-email" tooltip="Your Jira account email address">
        <Input
          id="config-email"
          onChange={onUsernameChange}
          value={jsonData.username || ''}
          placeholder="user@example.com"
          width={40}
        />
      </InlineField>
      <InlineField label="API Token" labelWidth={12} htmlFor="config-token" tooltip="Generate an API token from https://id.atlassian.com/manage-profile/security/api-tokens. Re-enter this if you change your email.">
        <SecretInput
          id="config-token"
          isConfigured={(secureJsonFields && secureJsonFields.token) as boolean}
          value={secureJsonData.token || ''}
          placeholder="secure json field (backend only)"
          width={40}
          onReset={onResetToken}
          onChange={onTokenChange}
        />
      </InlineField>
    </div>
  );
}
