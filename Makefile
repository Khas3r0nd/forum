utilize:
	@docker rmi -f $$(docker images -a -q)
	@docker rm $$(docker ps -a -f status=exited -q)	
image-creation:
	@time docker image build -f Dockerfile -t forum .
dockerize:
	@docker run -p 8081:8081 forum
