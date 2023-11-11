---
title: ホーム
---
# Hopefully Automatic Train Operation (HATO): 没入感を高める!? Nゲージ列車 全自動走行システム

![Hero](/assets/hero.png)

HATOは鉄道模型の世界に没入するためのNゲージ列車全自動運行システムです。
センサ群と電源装置からなる分散システムを独自プロトコル HLCP (Hato Line Control Protocol) を介して制御し、
複数の車両の位置や速度を管理することで事故を未然に防ぎ、車両を自動運行します。
更に、運転指令所にあるような列車制御UI等を実装することで、鉄道模型シミュレーションの没入度を高めます。
HATOは[GitHub](https://nyiyui/hato)にて公開しているopen sourceなプロジェクトです。
本プロジェクトは2023年度の未踏ジュニアに採択され、現在鋭意開発中です。

## Nゲージとは？
日本で一番よく売れている鉄道模型シリーズで、Nゲージの線路幅は9mm (NゲージのNはNineのNです) 、車両は本物の1/150の大きさです。

![Nゲージの線路と車両](/assets/9mm.png)

## 鉄道模型の楽しみ方
自分で好きなように鉄道が走る箱庭を作り、それを眺めて楽しむホビーです。

## HATOのシステム

Nゲージのシステムは、レールに電流を流すことで車両を動かします。HATOのメインプログラムは、そのレールに流れる電流の極性から車両の向きを、平均供給電圧値から車両のスピードを検知します。その他、車両の底面につけたRFIDタグを認識するセンサや、IR（赤外線）センサ、電流検出センサなどのセンサ群もモニターし、入力されたダイヤに沿いつつ事故を起こさないように、列車動力制御ドライバと通信して電流の極性や平均値を常時制御して車両を自動運行させます。

![HATOのシステム](/assets/system.png)

## デモ動画

<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/rcGFUpEQFpU?si=cXUUK7CVFKUmiYCf&amp;controls=0" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## [未踏ジュニア 成果報告会スライド](https://jr.mitou.org/projects/2023/hato)

<iframe src="https://docs.google.com/presentation/d/e/2PACX-1vTg74KgWRsJZkHOSdoS3f1Vs6Y6JPuo3XhNAyqh0CFVfhQ8ePn3AFxCCRjI8Nd3yi_bosN9fE0dCZWN/embed?start=false&loop=false&delayms=3000" frameborder="0" width="1920" height="1109" allowfullscreen="true" mozallowfullscreen="true" webkitallowfullscreen="true"></iframe>

<!--
## 今後の目標
-->
<!--
詳細は、こちらのスライドを参照してください。初めてのお披露は、11月3日の成果報告会になります。よろしければ、ぜひご参加ください。
-->
