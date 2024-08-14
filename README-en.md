---
title: Home (English)
---
# Hopefully Automatic Train Operation (HATO): Immersive Automatic Model Train Control System

![Hero](/assets/hero_en.png)

HATO is a fully automatic operation system for N gauge trains that aims to immerse users in the world of model railroading.
HATO controls a distributed system consisting of a group of sensors and power supply units via a custom protocol, HLCP (Hato Line Control Protocol), which is run over USB.
The system manages the position and speed of multiple trains while preventing accidents and automatically operating trains.
In addition, a train control interface similar to those found in an transit control centre enhances the immersiveness of your model train simulation.
HATO is an open source project available on [GitHub](https://nyiyui/hato). This project has been selected as an [Mitou Junior](https://jr.mitou.org/english/) project in 2023.

## What is N-gauge?
It is the best-selling model railroad series in Japan. The track width of N-gauge is 9 mm (9 = Nine = N-gauge), and the scale is 1/150 (for most Japanese models).

![N-Gage Tracks and Cars](/assets/9mm_en.png)

## How to enjoy model trains
This is a hobby in which you can create a box garden in which trains run as you like and enjoy looking at them.

## HATO system overview
HATO's main program detects the direction of the train based on the polarity of the current flowing on the rails, and the speed of the train based on the average supply voltage value. It also monitors other sensors such as RFID tags attached to the bottom of the car, IR sensors, and current detection sensors, and communicates with the train power control driver ("kdss") to control the polarity and average value of the current at all times to keep the car running automatically according to a set schedule while avoiding accidents.

![HATO system overview](/assets/system_en.png)

## Demo Video

<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/feEj1TXzvtw?si=cXUUK7CVFKUmiYCf&amp;controls=0" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## [Mitou Junior Mentorship Program](https://jr.mitou.org/english/) Final Presentation Slides

<iframe src="https://docs.google.com/presentation/d/e/2PACX-1vTna_kV5mdMgtzeJbguJie78pdeHvQdgSHAvareKW4sNdjQMN_z_cBeJy-oTP5OM6jKveWvjQBx_t1l/embed?start=false&loop=false&delayms=3000" frameborder="0" width="960" height="569" allowfullscreen="true" mozallowfullscreen="true" webkitallowfullscreen="true"></iframe>
