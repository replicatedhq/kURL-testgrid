import * as React from "react";
import * as moment from "moment";

import { Link } from "react-router-dom";

import "../../assets/scss/components/RunTable.scss";

export default class RunTable extends React.Component {
    cancelRun = async (run) => {
        if (!window.confirm(`Cancel run ${run.id}?\n\nPending instances will be skipped and running VMs will be terminated on the runner hosts.`)) {
            return;
        }

        try {
            const res = await fetch(`${window.env.API_ENDPOINT}/ref/${run.id}/cancel`, {
                method: "POST",
            });

            if (!res.ok) {
                throw new Error(`cancel failed with status ${res.status}`);
            }

            if (this.props.onRunCancelled) {
                this.props.onRunCancelled();
            }
        } catch(err) {
            console.error(err);
            window.alert(`Failed to cancel run: ${err.message}`);
        }
    }

    render() {
        const rows = this.props.runs.map((run) => {
            return (
                <tr key={run.id}>
                    <td><Link to={`/run/${run.id}`}>{run.id}</Link></td>
                    <td>{moment(run.created_at).format("MMM D, YYYY h:mma")}</td>
                    <td>{run.last_start && moment(run.last_start).format("MMM D, YYYY h:mma")}</td>
                    <td>{run.last_response && moment(run.last_response).format("MMM D, YYYY h:mma")}</td>
                    <td>{`${run.success_count}`}</td>
                    <td>{`${run.failure_count}`}</td>
                    <td>{`${run.running_runs}`}</td>
                    <td>{`${run.pending_runs}`}</td>
                    <td>
                        {(run.running_runs > 0 || run.pending_runs > 0) &&
                            <button type="button" className="btn xsmall secondary red" onClick={() => this.cancelRun(run)}>Cancel</button>
                        }
                    </td>
                </tr>
            )
        })
        return (
            <table>
                <thead>
                    <tr>
                        <th>kURL Ref</th>
                        <th>Run Started At</th>
                        <th>Last Instance Started At</th>
                        <th>Last Instance Completed At</th>
                        <th>Successes</th>
                        <th>Failures</th>
                        <th>Running</th>
                        <th>Pending</th>
                        <th></th>
                    </tr>
                </thead>
                <tbody>
                    {rows}
                </tbody>
            </table>
        );
    }
}
