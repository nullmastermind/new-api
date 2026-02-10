set windows-shell := ["pwsh.exe", "-NoLogo", "-Command"]

Auggie-Code:
    auggie

docker-build:
    docker build -t nullmastermind/new-api:latest .

docker-push: docker-build
    docker push nullmastermind/new-api:latest

docker-build-push: docker-push

pull:
    auggie command git-pull "pull & merge remore 'fork', brain 'main'" --print --compact
