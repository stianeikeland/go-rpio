#!/bin/bash

echo "Setting Pin 11 to High!"
./gpiocli -p 11 -s=+1

if ./gpiocli -p 11 -l=true  ; then
    echo "Pin 11 is High!"
  else
    echo "Pin 11 is Low"
fi

sleep 1

echo "Setting Pin 11 to Low!"
./gpiocli -p 11 -s=-1

if ./gpiocli -p 11 -l=true  ; then
    echo "Pin 11 is High!"
  else
    echo "Pin 11 is Low"
fi

