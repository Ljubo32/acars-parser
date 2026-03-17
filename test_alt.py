#!/usr/bin/env python3
# Test altitude decoding for ADS-C

hex_data = "071F30C859D98908D9B59D21EC"
data_bytes = bytes.fromhex(hex_data)
basic_data = data_bytes[1:11]

print("=== ADS-C Altitude Decoding Analysis ===")
print(f"Hex data: {hex_data}")
print(f"Basic report (10 bytes): {basic_data.hex().upper()}")
print()

# Read all 80 bits
bits_80 = int.from_bytes(basic_data, 'big')
print(f"80-bit value: {bits_80:080b}")
print()

# Extract fields according to ARINC 622 format:
# Lat: bits 0-20 (21 bits)
# Lon: bits 21-41 (21 bits)  
# Alt: bits 42-57 (16 bits)
# Time: bits 58-72 (15 bits)
# Flags: bits 73-79 (7 bits)

lat_raw = (bits_80 >> 59) & 0x1FFFFF
lon_raw = (bits_80 >> 38) & 0x1FFFFF
alt_raw = (bits_80 >> 22) & 0xFFFF
time_raw = (bits_80 >> 7) & 0x7FFF
flags = bits_80 & 0x7F

print(f"Latitude raw: {lat_raw} (0x{lat_raw:05X})")
print(f"Longitude raw: {lon_raw} (0x{lon_raw:05X})")
print(f"Altitude raw: {alt_raw} (0x{alt_raw:04X})")
print(f"Time raw: {time_raw} -> {time_raw * 0.125:.3f} sec")
print(f"Flags: {flags:07b}")
print()

# Test different altitude formulas
print("=== Altitude Calculations ===")
print(f"Current implementation: raw * 2 + 20000 = {alt_raw * 2 + 20000} ft")

# Check if signed
if alt_raw & 0x8000:
    alt_signed = alt_raw - 65536
    print(f"If signed: {alt_signed} * 2 + 20000 = {alt_signed * 2 + 20000} ft")
else:
    print(f"Not signed (bit 15 = 0)")

print(f"Expected (acars lib): 37004 ft")
print()

# What raw value gives 37004?
target = 37004
needed_raw = (target - 20000) // 2
print(f"To get 37004 ft, need raw = {needed_raw} (0x{needed_raw:04X})")
print(f"Difference: {alt_raw} - {needed_raw} = {alt_raw - needed_raw}")
print()

# Check if there's an off-by-one in bit position
print("=== Testing alternative bit positions ===")
for offset in range(-2, 3):
    shift = 22 + offset
    if shift >= 0:
        test_alt = (bits_80 >> shift) & 0xFFFF
        calc_ft = test_alt * 2 + 20000
        print(f"Bits {42-offset}-{57-offset} (shift {shift}): {test_alt} -> {calc_ft} ft", end="")
        if calc_ft == 37004:
            print(" *** MATCH ***")
        else:
            print()

print()

# Let's also check if the libacars uses a different base
print("=== Testing different formulas ===")
# Maybe it's altitude / 4 ft per unit?
print(f"Formula: raw * 4 + 20000 = {alt_raw * 4 + 20000} ft")
print(f"Formula: raw + 20000 = {alt_raw + 20000} ft")
print(f"Formula: raw * 2 = {alt_raw * 2} ft")
print(f"Formula: raw * 4 = {alt_raw * 4} ft")
