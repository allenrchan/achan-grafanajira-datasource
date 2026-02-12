FROM alpine:latest
COPY dist /plugins/grafana-jira-datasource
CMD ["/bin/sh", "-c", "cp -r /plugins/grafana-jira-datasource /var/lib/grafana/plugins/"]
