import { QueryEditorHelpProps } from '@grafana/data';
import {JiraQuery, METRICS} from "../../types";
import React from "react";

export default function QueryEditorHelp(props: QueryEditorHelpProps<JiraQuery>): any {
    const examples = [
        {
            title: 'cycle time scatterplot',
            expression: "project = 'MAEN'",
            metric: METRICS.CYCLE_TIME,
            label: '',
            quantile: 85
        },
    ];

    return (
        <div>
            <h2>Cheat Sheet</h2>
            {examples.map((item, index) => (
                <div className="cheat-sheet-item" key={index} onClick={(e) => props.onClickExample({ refId: 'A', jqlQuery: item.expression, metric: item.metric, quantile: item.quantile } as JiraQuery)}>
                    <div className="cheat-sheet-item__title" >{item.title}</div>
                    <div className="cheat-sheet-item__label">{item.label}</div>
                </div>
            ))}
        </div>
    );
};
