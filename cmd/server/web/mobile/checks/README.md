# Mock Check Assets

The mobile app uses these files (already present):

| Check | Scenario        | Front             | Back             |
|-------|-----------------|-------------------|------------------|
| 1     | Clean pass      | clean-check.png   | back-of-check.png |
| 2     | IQA Blur        | blurry-check.png  | back-of-check.png |
| 3     | IQA Glare       | glare-check.png   | back-of-check.png |
| 4     | MICR fail       | micr-check.png    | back-of-check.png |
| 5     | Amount mismatch | mismatch-check.png| back-of-check.png |
| 6     | Duplicate       | clean-check.png   | back-of-check.png |

**Formats:** PNG or JPEG

Each check selection in the carousel triggers the corresponding vendor stub scenario.
