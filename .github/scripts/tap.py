"""Taps the on-screen element with this text or accessibility label, on the connected emulator."""
import re
import subprocess
import sys
import xml.etree.ElementTree as ET

label = sys.argv[1]
subprocess.run(["adb", "shell", "uiautomator", "dump", "/sdcard/ui.xml"], check=True, stdout=subprocess.DEVNULL)
xml = subprocess.run(["adb", "exec-out", "cat", "/sdcard/ui.xml"], check=True, capture_output=True, text=True).stdout
for node in ET.fromstring(xml).iter("node"):
    if label in (node.get("text"), node.get("content-desc")):
        x1, y1, x2, y2 = map(int, re.findall(r"\d+", node.get("bounds")))
        subprocess.run(["adb", "shell", "input", "tap", str((x1 + x2) // 2), str((y1 + y2) // 2)], check=True)
        break
else:
    sys.exit(f"nothing on screen says {label!r}")
