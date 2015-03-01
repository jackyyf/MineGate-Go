# MineGate-Go [![Build Status](https://travis-ci.org/jackyyf/MineGate-Go.svg?branch=master)](https://travis-ci.org/jackyyf/MineGate-Go)

  Minegate is reverse proxy for minecraft, focused on providing the missing 
virtual host functionality, which is supported by the protocol.

## Usage

  Please refer to config.yml for configuration. Place config.yml within
current directory, and minegate will load it automatically. Custom 
config file support will be added soon.

  Feature requests are welcome via issues.

## Branches

  [master](tree/master/): the development branch, code within master are likely to be broken.  
  [stable](tree/stable/): the last successful build, compiled and tested with basic tests, expect to work.  
  [release](tree/release/): the latest release, passed all tests in stable, plus manual feature tests.  
  
  You should only use release versions in production environments!
