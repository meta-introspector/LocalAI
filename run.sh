
export IMAGE="quay.io/go-skynet/local-ai:master-cublas-cuda12"
docker kill localai
docker rm localai
docker run --name localai -p 8080:8080 -v $PWD/models:/models -ti      --env-file .envfile    -v $PWD/logs:/app/logs     --rm $IMAGE --models-path /models --context-size 700 --threads 4
