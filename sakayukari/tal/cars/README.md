# 編成管理システム

## 型

```yaml
sets:
  e5f6bb45-0abe-408c-b8e0-e2772f3bbdb0:
    comment: "E233系 E-66編成"
    length: 45000 # µm
    base-velocity:
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
