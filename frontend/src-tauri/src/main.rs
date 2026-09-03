// Evita una consola adicional en compilaciones de Windows.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    router_core_desktop_lib::run();
}
