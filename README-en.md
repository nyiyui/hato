---
title: Home
---
# Hopefully Automatic Train Operation (HATO): Immersive Automatic Model Train Control System

![Hero](/assets/hero.png)

HATO is a fully automatic operation system for N gauge trains that immerses you in the world of model railroading.
It controls a distributed system consisting of a group of sensors and power supply units via a proprietary protocol, HLCP (Hato Line Control Protocol),
The system manages the position and speed of multiple trains to prevent accidents and automatically operate trains.
In addition, the implementation of a train control UI and other features similar to those found in an operating control center will enhance the immersiveness of the model train simulation.
HATO is an open source project available on [GitHub](https://nyiyui/hato).This project has been selected as an Mitou junior project in 2023 and is currently under development.

## What is N-Gage?
It is the best-selling model railroad series in Japan. The track width of N gauge is 9mm (N in N gauge is N in Nine), and the cars are 1/150th the size of the real thing.

![N-Gage Tracks and Cars](/assets/9mm.png)

## How to enjoy model trains
This is a hobby in which you can create a box garden in which trains run as you like and enjoy looking at them.

## HATO system overview

HATO's main program detects the direction of the train based on the polarity of the current flowing on the rails, and the speed of the train based on the average supply voltage value. It also monitors other sensors such as RFID tags attached to the bottom of the car, IR (infrared) sensors, and current detection sensors, and communicates with the train power control driver to control the polarity and average value of the current at all times to keep the car running automatically according to the input schedule and avoid causing accidents.

![HAT system overview](/assets/system.png)

## Demo Video

<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/rcGFUpEQFpU?si=cXUUK7CVFKUmiYCf&amp;controls=0" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

<!--
## 今後の目標
-->
<!--
詳細は、こちらのスライドを参照してください。初めてのお披露は、11月3日の成果報告会になります。よろしければ、ぜひご参加ください。
-->
