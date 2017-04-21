# Vanir

Vanir is a database masking tool.

    $ cat example.sql
    INSERT INTO `users` (name, email) VALUES ("Fumiaki MATSUSHIMA", "mtsmfm@gmail.com");
    $ cat example.yml
    users: # Table name
      name: "AAA" # Simply replace all name with "AAA"
      email: "{{.First 6}}.{{.Hashed}}.{{.Last 3}}@example.com" # You can use go template here
    $ cat example.sql | vanir --config example.yml > masked.sql
    Masking `users`...
    $ cat masked.sql
    insert into users(name, email) values ('AAA', 'mtsmfm.hr9FYQ5ZbHCEonO9ucjozR6DhBzyPwW.com@example.com');

## Usage

1. Write config file:

  ```yaml
  users: # Table name
    name: "AAA" # Simply replace all name with "AAA"
    email: "{{.Raw}}+{{.Hashed}}@example.com" # You can use go template here
  ```

2. Pipe the dumped SQL:

At this time, vanir can mask SQL dumped with `--complete-insert` option only.

        mysqldump --complete-insert | vanir -c path/to/config.yml
        # or
        mysqldump --complete-insert > dump.sql
        cat dump.sql | vanir -c path/to/config.yml

## Installation

Donwload an executable binary for your platform:

https://github.com/mtsmfm/vanir/releases

## Development

### Requirements

- docker
- docker-compose

### Setup

    git clone https://github.com/mtsmfm/vanir
    cd vanir
    docker-compose run app go-wrapper download

### Compile

    docker-compose run app go-wrapper install

### Run

    docker-compose run app go-wrapper run
