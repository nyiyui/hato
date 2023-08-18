# さかゆかり / Sakayukari

- train/列車: 一編成の一運用を指す。（増解結は新しい列車をつくる。）
- line/domain/領域：独立してduty cycleを調節出来る線区。

## `cars.yaml`

`cars.yaml`には、編成に関するデータを用意します。
(わかり易いのでYAMLを使いますが、最終的にはJSONになれば何でもOKです。)

1. 適当なUUIDを決める(編成をこれで参照するので、かぶらないように！)
2. duty cycleに対しての編成のモータ車全ての速度
  1. 直線でduty cycleに対しての速度を測り、結果を線形関数に近似する
  2. y=mx+bにし、`base-velocity`に入れる
3. 車両ごとに
  1. 長さを測る
  2. 大電流(~0.7Vで200µA以上)を使うのかをもとめる
  3. RFIDタグがあれば、MifareのUIDとタグの場所を記載

```yaml
sets:
  e5f6bb45-0abe-408c-b8e0-e2772f3bbdb0:
    comment: "E233系 E-66編成"
    length: 45000 # µm
    base-velocity: # speed = m(duty cycle)+b
      m: 6180 # µm/duty cycle
      b: -34574 # µm
    cars:
      - comment: "クハE233-3516 (15号車)"
        large-current: false # (平均)0.7Vでは約4µA
        length: 131000 # µm
        mifare-id: 003b3712000003 # RFIDタグの4/7バイトID
        mifare-pos: 73000 # µm 車両A側からB側に向かってRFIDタグアンテナ中央
        // -snip-
      - comment: "モハE233-3616 13号車"
        large-current: true # (平均)0.7Vでは約200µA
        length: 131000 # µm
        mifare-id: 002d7912000003
        mifare-pos: 61000
```
