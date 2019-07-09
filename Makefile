init:
	go get -v
	go get github.com/cespare/reflex

dev:
	reflex -d none -s -R vendor. -r '\.(go|yml)$$' -- go run .

build:
	cd ~/WebstormProjects/floatdream-frontend/; yarn deploy
	-rm floatdream
	go build -o floatdream
	rice append --exec floatdream
