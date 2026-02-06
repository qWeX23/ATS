You are deciding a single trade action for the next bar.

{{- if .Context }}
Context:
{{ .Context }}
{{- end }}

Market snapshot:
timestamp={{ .Timestamp }}
close={{ printf "%.4f" .Close }}
sma={{ printf "%.4f" .SMA }}
position_qty={{ .PositionQty }}
max_qty={{ .MaxQty }}

Call the decide_trade tool with a single action, qty, and reason.
