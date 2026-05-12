set windows-shell := ["pwsh.exe", "-NoLogo", "-Command"]

Auggie-Code:
    auggie

commit := `git rev-parse --short HEAD`

docker-build:
    docker build -t nullmastermind/new-api:latest -t nullmastermind/new-api:{{commit}} .

docker-push: docker-build
    docker push nullmastermind/new-api:latest
    docker push nullmastermind/new-api:{{commit}}

docker-build-push: docker-push

pull:
    auggie command git-pull "Pull and merge the remote 'fork', branch 'main', then run `just docker-build` to ensure the image is built successfully." --print --compact
