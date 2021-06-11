PKG_NAME=$(shell cat go.mod | head -n 1 | awk '{print $$2}')

up: 
	@docker-compose up -d

down:
	@docker-compose down

doc:
	@-pkill godoc
	@godoc &
	@sleep 0.5
	@open "http://127.0.0.1:6060/pkg/$(PKG_NAME)"

dev-center:
	@air -c .center.toml

dev-edge:
	@air -c .edge.toml