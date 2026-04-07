import Quickshell
import Quickshell.Hyprland
import Quickshell.Services.SystemTray
import Quickshell.Io
import QtQuick
import QtQuick.Layouts

PanelWindow {
    id: bar

    // ── Screen routing ──
    readonly property bool isMain: screen.name === "DVI-I-1"
    readonly property var wsIds: isMain ? [1, 2, 3, 4, 5, 6, 7] : [8, 9, 10]
    property var hyprMonitor: Hyprland.monitorFor(screen)
    property int activeWsId: hyprMonitor?.activeWorkspace?.id ?? -1

    // ── Theme (matches waybar/style.css warm-dark palette) ──
    readonly property color bgPill:          Qt.rgba(0.165, 0.145, 0.125, 0.78)
    readonly property color borderPill:      Qt.rgba(0.859, 0.686, 0.431, 0.08)
    readonly property color textPrimary:     "#e6ddd4"
    readonly property color textDim:         "#968b80"
    readonly property color accentBg:        Qt.rgba(0.859, 0.659, 0.420, 0.28)
    readonly property color accentBorder:    Qt.rgba(0.859, 0.686, 0.431, 0.12)
    readonly property color highlightBg:     Qt.rgba(0.859, 0.659, 0.420, 0.24)
    readonly property color highlightBorder: Qt.rgba(0.859, 0.659, 0.420, 0.30)
    readonly property color highlightHover:  Qt.rgba(0.859, 0.659, 0.420, 0.38)
    readonly property color hoverBg:         Qt.rgba(0.859, 0.686, 0.431, 0.12)
    readonly property color sepColor:        Qt.rgba(0.859, 0.686, 0.431, 0.06)
    readonly property color cpuColor:        "#7fb4ca"
    readonly property color memColor:        "#d2a8c0"
    readonly property color tempColor:       "#a8c686"
    readonly property color tempCritColor:   "#f28b82"
    readonly property color audioColor:      "#7aa89f"
    readonly property color netColor:        "#98bb6c"

    // ── Live state ──
    property string cpuVal:  "0"
    property string memVal:  "0"
    property string tempVal: "0"
    property string volVal:  "0"
    property bool   volMute: false
    property string netVal:  "󰖪"
    property string clockVal: ""
    property bool   clockAltMode: false
    property var    prevCpu: null       // delta-based CPU

    // ── Panel geometry (same as waybar) ──
    anchors { top: true; left: true; right: true }
    margins { top: 6; left: 8; right: 8 }
    height: 26
    color: "transparent"

    // ── Reusable pill background ──
    component Pill: Rectangle {
        color: bar.bgPill
        border { width: 1; color: bar.borderPill }
        radius: 14
        height: 26
    }
    component Sep: Rectangle {
        width: 1; height: 14; color: bar.sepColor
        Layout.leftMargin: 8; Layout.rightMargin: 8
    }

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    //  CONTENT
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    Item {
        anchors.fill: parent

        // ── LEFT ──
        Row {
            anchors { left: parent.left; verticalCenter: parent.verticalCenter }
            spacing: 4

            // Launcher button
            Pill {
                visible: bar.isMain
                width: launcherIcon.implicitWidth + 24
                color: launcherMa.containsMouse ? bar.highlightHover : bar.highlightBg
                border.color: bar.highlightBorder

                Text {
                    id: launcherIcon; anchors.centerIn: parent
                    text: "󱄅"
                    color: "#faf4ee"
                    font { family: "CaskaydiaCove Nerd Font Mono"; pixelSize: 18 }
                }
                MouseArea {
                    id: launcherMa; anchors.fill: parent
                    hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                    onClicked: Hyprland.dispatch("exec", "~/.config/hypr/scripts/rofi-launcher.sh")
                }
                Behavior on color { ColorAnimation { duration: 200 } }
            }

            // Workspaces
            Pill {
                width: wsRow.implicitWidth + 12

                Row {
                    id: wsRow; anchors.centerIn: parent; spacing: 2

                    Repeater {
                        model: bar.wsIds

                        Rectangle {
                            required property int modelData
                            property bool active: bar.activeWsId === modelData
                            property bool occupied: {
                                var ws = Hyprland.workspaces.values;
                                for (var i = 0; i < ws.length; i++)
                                    if (ws[i].id === modelData) return true;
                                return false;
                            }

                            width: active ? 38 : 28; height: 22; radius: 12
                            color: active ? bar.accentBg
                                         : wsMa.containsMouse ? bar.hoverBg
                                         : "transparent"
                            border { width: active ? 1 : 0; color: bar.accentBorder }
                            opacity: (!active && !occupied) ? 0.5 : 1.0

                            Text {
                                anchors.centerIn: parent
                                text: parent.modelData
                                color: parent.active ? "#faf4ee" : bar.textDim
                                font { family: "Noto Sans"; pixelSize: 13 }
                            }
                            MouseArea {
                                id: wsMa; anchors.fill: parent
                                hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                                onClicked: Hyprland.dispatch("workspace", parent.modelData.toString())
                            }

                            Behavior on width   { NumberAnimation { duration: 200 } }
                            Behavior on color   { ColorAnimation  { duration: 200 } }
                            Behavior on opacity { NumberAnimation { duration: 200 } }
                        }
                    }
                }

                // scroll to switch workspace
                WheelHandler {
                    onWheel: function(event) {
                        Hyprland.dispatch("workspace", event.angleDelta.y > 0 ? "e-1" : "e+1")
                    }
                }
            }

            // Window title
            Pill {
                visible: bar.isMain && winLabel.text !== ""
                width: Math.min(winLabel.implicitWidth + 20, 360)
                clip: true

                Text {
                    id: winLabel; anchors.centerIn: parent
                    width: parent.width - 20
                    text: Hyprland.activeWindow?.title ?? ""
                    color: bar.textDim
                    font { family: "Noto Sans"; pixelSize: 13 }
                    elide: Text.ElideRight; maximumLineCount: 1
                }
            }
        }

        // ── CENTER ──
        Pill {
            visible: bar.isMain
            anchors.centerIn: parent
            width: clockLabel.implicitWidth + 24

            Text {
                id: clockLabel; anchors.centerIn: parent
                text: bar.clockVal; color: bar.textPrimary
                font { family: "Noto Sans"; pixelSize: 14; weight: Font.DemiBold }
            }
            MouseArea {
                anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                onClicked: bar.clockAltMode = !bar.clockAltMode
            }
        }

        // ── RIGHT ──
        Row {
            visible: bar.isMain
            anchors { right: parent.right; verticalCenter: parent.verticalCenter }
            spacing: 4

            // Hardware group: CPU | Memory | Temperature
            Pill {
                width: hwRow.implicitWidth + 20
                RowLayout {
                    id: hwRow; anchors.centerIn: parent; spacing: 0

                    Text {
                        text: "󰻠 " + bar.cpuVal + "%"
                        color: bar.cpuColor; font.pixelSize: 13
                    }
                    Sep {}
                    Text {
                        text: "󰍛 " + bar.memVal + "%"
                        color: bar.memColor; font.pixelSize: 13
                    }
                    Sep {}
                    Text {
                        property int t: parseInt(bar.tempVal) || 0
                        text: (t >= 80 ? "󰹕" : t >= 60 ? "󰹔" : t >= 40 ? "󰔏" : "󰹖")
                              + " " + bar.tempVal + "°C"
                        color: t >= 80 ? bar.tempCritColor : bar.tempColor
                        font.pixelSize: 13
                    }
                }
            }

            // Status group: Audio | Network
            Pill {
                width: stRow.implicitWidth + 20
                RowLayout {
                    id: stRow; anchors.centerIn: parent; spacing: 0

                    Text {
                        text: {
                            if (bar.volMute) return "󰝟 mute";
                            var v = parseInt(bar.volVal) || 0;
                            var icon = v > 50 ? "󰕾"
                                     : v > 0  ? "󰖀"
                                     :           "󰕿";
                            return icon + " " + bar.volVal + "%";
                        }
                        color: bar.audioColor; font.pixelSize: 13

                        MouseArea {
                            anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                            onClicked: Hyprland.dispatch("exec", "pavucontrol")
                        }
                    }
                    Sep {}
                    Text {
                        text: bar.netVal; color: bar.netColor; font.pixelSize: 13
                    }
                }
            }

            // System tray
            Pill {
                visible: trayRep.count > 0
                width: trayRow.implicitWidth + 16

                Row {
                    id: trayRow; anchors.centerIn: parent; spacing: 6

                    Repeater {
                        id: trayRep
                        model: SystemTray.items

                        Image {
                            required property var modelData
                            source: modelData.icon ?? ""
                            width: 15; height: 15
                            fillMode: Image.PreserveAspectFit

                            MouseArea {
                                anchors.fill: parent
                                acceptedButtons: Qt.LeftButton | Qt.RightButton
                                onClicked: function(mouse) {
                                    if (mouse.button === Qt.RightButton)
                                        modelData.secondaryActivate();
                                    else
                                        modelData.activate();
                                }
                            }
                        }
                    }
                }
            }

            // Power button
            Pill {
                width: powerIcon.implicitWidth + 24
                color: powerMa.containsMouse ? bar.highlightHover : bar.highlightBg
                border.color: bar.highlightBorder

                Text {
                    id: powerIcon; anchors.centerIn: parent
                    text: "󰐥"
                    color: "#faf4ee"
                    font { family: "CaskaydiaCove Nerd Font Mono"; pixelSize: 18 }
                }
                MouseArea {
                    id: powerMa; anchors.fill: parent
                    hoverEnabled: true; cursorShape: Qt.PointingHandCursor
                    onClicked: Hyprland.dispatch("exec", "~/.config/hypr/scripts/power-menu.sh")
                }
                Behavior on color { ColorAnimation { duration: 200 } }
            }
        }
    }

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    //  CLOCK TIMER (1 s)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    Timer {
        interval: 1000; running: true; repeat: true; triggeredOnStart: true
        onTriggered: {
            var d = new Date();
            var ws = ["周日","周一","周二","周三","周四","周五","周六"];
            var wl = ["星期日","星期一","星期二","星期三","星期四","星期五","星期六"];
            var mm = String(d.getMonth() + 1).padStart(2, "0");
            var dd = String(d.getDate()).padStart(2, "0");
            var hh = String(d.getHours()).padStart(2, "0");
            var mi = String(d.getMinutes()).padStart(2, "0");

            bar.clockVal = bar.clockAltMode
                ? "󰃭 " + d.getFullYear() + "年" + mm + "月" + dd + "日 " + wl[d.getDay()]
                : "󰥔 " + mm + "月" + dd + "日 " + hh + ":" + mi + " " + ws[d.getDay()];
        }
    }

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    //  SYSTEM STATS (3 s, main bar only)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    Process {
        id: cpuProc
        command: ["bash", "-c", "awk '/^cpu / {print $2,$3,$4,$5,$6,$7,$8}' /proc/stat"]
        stdout: SplitParser {
            onRead: function(data) {
                var p = data.trim().split(/\s+/).map(Number);
                var active = p[0] + p[1] + p[2] + p[4] + p[5] + p[6];
                var total  = active + p[3];
                if (bar.prevCpu) {
                    var dt = total - bar.prevCpu.total;
                    bar.cpuVal = dt > 0 ? Math.round((active - bar.prevCpu.active) / dt * 100).toString() : "0";
                }
                bar.prevCpu = { active: active, total: total };
            }
        }
    }

    Process {
        id: memProc
        command: ["bash", "-c", "awk '/MemTotal/{t=$2} /MemAvailable/{printf \"%.0f\",(t-$2)/t*100}' /proc/meminfo"]
        stdout: SplitParser { onRead: function(data) { bar.memVal = data.trim() } }
    }

    Process {
        id: tempProc
        command: ["bash", "-c", "cat /sys/devices/platform/coretemp.0/hwmon/hwmon*/temp1_input 2>/dev/null | head -1 | awk '{printf \"%.0f\",$1/1000}'"]
        stdout: SplitParser { onRead: function(data) { bar.tempVal = data.trim() } }
    }

    Process {
        id: volProc
        command: ["bash", "-c", "wpctl get-volume @DEFAULT_AUDIO_SINK@"]
        stdout: SplitParser {
            onRead: function(data) {
                // "Volume: 0.55" or "Volume: 0.55 [MUTED]"
                var parts = data.trim().split(/\s+/);
                if (parts.length >= 2) {
                    bar.volVal  = Math.round(parseFloat(parts[1]) * 100).toString();
                    bar.volMute = data.indexOf("[MUTED]") !== -1;
                }
            }
        }
    }

    Process {
        id: netProc
        command: ["bash", "-c",
            "ip -j addr show 2>/dev/null | jq -r " +
            "'[.[]|select(.operstate==\"UP\" and .ifname!=\"lo\")] " +
            "| if length==0 then \"disconnected\" " +
            "  else .[0] | .ifname + \" \" + (.addr_info[]|select(.family==\"inet\")|.local) " +
            "  end'"]
        stdout: SplitParser {
            onRead: function(data) {
                var d = data.trim();
                if (d === "" || d === "disconnected") {
                    bar.netVal = "󰖪";
                } else if (d.startsWith("wl")) {
                    bar.netVal = "󰖩 " + d.split(" ")[1];
                } else {
                    bar.netVal = "󰈀 " + d.split(" ")[1];
                }
            }
        }
    }

    Timer {
        interval: 3000; running: bar.isMain; repeat: true; triggeredOnStart: true
        onTriggered: {
            cpuProc.running  = true;
            memProc.running  = true;
            tempProc.running = true;
            volProc.running  = true;
            netProc.running  = true;
        }
    }
}
