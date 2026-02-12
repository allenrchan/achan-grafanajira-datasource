import { useAsync } from 'react-use';
import type { SelectableValue } from '@grafana/data';
import {DataSource} from "../../datasource";
import {JiraQuery} from "../../types";

type AsyncQueryTypeState = {
  loading: boolean;
  queryTypes: Array<SelectableValue<string>>;
  error: Error | undefined;
};

export function useMetricTypes(datasource: DataSource): AsyncQueryTypeState {
  const result = useAsync(async () => {
    const { queryTypes } = await datasource.getAvailableMetricTypes();
    return queryTypes
  }, [datasource]);

  return {
    loading: result.loading,
    queryTypes: result.value ?? [],
    error: result.error,
  };
}
