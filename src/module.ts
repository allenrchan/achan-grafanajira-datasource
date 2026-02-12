import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor/QueryEditor';
import { JiraQuery, MyDataSourceOptions } from './types';
import QueryEditorHelp from "./components/QueryEditor/QueryEditorHelp";

export const plugin = new DataSourcePlugin<DataSource, JiraQuery, MyDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor)
    .setQueryEditorHelp(QueryEditorHelp);
