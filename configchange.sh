
	config='{"contexts":{"local":{"endpoint":"http://$ENDPOINT","token":"Default","platform":"kubernetes","adapters":[],"channel":"stable","version":"latest"}},"current-context":"local","tokens":[{"location":"auth.json","name":"Default"}]}'

	echo $config | yq e '."contexts"."local"."adapters"[0]="'$1'"' -P - > ~/.meshery/config.yaml

	cat ~/.meshery/config.yaml