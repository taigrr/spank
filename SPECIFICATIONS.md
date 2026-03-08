# Spank Windows Specifications

This document outlines the required hardware and software specifications for running `spank` on Windows.

## 1. Hardware Requirements

### Mandatory

- **3D Accelerometer (IMU)**: The device must have a built-in accelerometer that is recognized by the Windows Sensor API.
  - _Common in_: Microsoft Surface devices, 2-in-1 laptops, tablets, and many modern Ultrabooks.
  - _Commonly missing in_: Most gaming laptops, older budget laptops, and desktop PCs.

### Performance

- **Polling Rate Support**: The sensor should ideally support a data updated rate of at least 100Hz (10ms) for responsive slapping detection.

## 2. Software Requirements

### Operating System

- **Windows 10 or 11**: Required for the Windows Sensor API (COM-based) to function correctly.

### Driver Support

- **HID Sensor Drivers**: Ensure that "HID Sensor Collection" or similar entries are present and working in Device Manager under "Sensors".

### Build / Development

- **Go 1.26+**: Required for building from source.
- **GCC / MinGW**: (If using CGO components, though currently planned as pure Go via OLE).

## 3. Technical Implementation Details

### Detection Thresholds (The "Bar")

- **Minimum Amplitude**: 0.05g (default).
- **Default Polling**: 10ms.
- **Normalization**: All sensor data is normalized to g-force before being processed by the detection logic to ensure parity with the macOS version.

## 4. Verification Check

To verify if your system is compatible, you can run:

```bash
spank --status
```

_(Feature in progress: will report sensor availability and current G-force readings)_
