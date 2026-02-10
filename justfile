set windows-shell := ["pwsh.exe", "-NoLogo", "-Command"]

Auggie-Code:
    auggie

docker-build:
    docker build -t nullmastermind/new-api:latest .

docker-push: docker-build
    docker push nullmastermind/new-api:latest

docker-build-push: docker-push

pull:
    auggie command git-pull "Pull and merge the remote 'fork', branch 'main', then run `just docker-build` to ensure the image is built successfully." --print --compact
