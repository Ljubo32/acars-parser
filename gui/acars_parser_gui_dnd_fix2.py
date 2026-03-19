"""ACARS Parser — GUI (extract) with Drag & Drop + auto output naming

Drag & drop support uses tkinterdnd2.

Install:
  pip install tkinterdnd2
"""
import os
import threading
import subprocess
import json
import tempfile
import tkinter as tk
from tkinter import ttk, filedialog, messagebox

_HAS_DND = False
try:
    # IMPORTANT: TkinterDnD is a MODULE; the actual window base class is TkinterDnD.Tk
    from tkinterdnd2 import DND_FILES, TkinterDnD  # type: ignore
    _HAS_DND = True
except Exception:
    DND_FILES = None
    TkinterDnD = None  # type: ignore


def _split_dnd_files(data: str):
    """Split the string from tkinterdnd2 into file paths."""
    if not data:
        return []
    data = data.strip()
    out = []
    cur = ""
    in_brace = False
    for ch in data:
        if ch == "{":
            in_brace = True
            cur = ""
        elif ch == "}":
            in_brace = False
            if cur:
                out.append(cur)
                cur = ""
        elif ch.isspace() and not in_brace:
            if cur:
                out.append(cur)
                cur = ""
        else:
            cur += ch
    if cur:
        out.append(cur)

    # Normalize slashes (some setups deliver escaped backslashes)
    out = [p.replace('\\\\', '\\') for p in out]
    return out


BaseTk = TkinterDnD.Tk if _HAS_DND else tk.Tk  # type: ignore[attr-defined]


class App(BaseTk):
    def __init__(self):
        super().__init__()
        self.title("ACARS Parser — GUI (extract) • Drag & Drop")
        self.geometry("1050x720")

        self.exe_var = tk.StringVar(value=r".\acars_parser.exe")
        self.outdir_var = tk.StringVar(value="")  # empty => same folder as input
        self.format_var = tk.StringVar(value="json")
        self.pretty_var = tk.BooleanVar(value=True)
        self.all_var = tk.BooleanVar(value=True)
        self.stats_var = tk.BooleanVar(value=False)
        self.merge_var = tk.BooleanVar(value=False)

        self._build_ui()

        if _HAS_DND:
            # Drop onto the whole window
            self.drop_target_register(DND_FILES)  # type: ignore[attr-defined]
            self.dnd_bind("<<Drop>>", self._on_drop)  # type: ignore[attr-defined]

    def _build_ui(self):
        frm = ttk.Frame(self, padding=10)
        frm.pack(fill="both", expand=True)

        row0 = ttk.Frame(frm)
        row0.pack(fill="x", pady=(0, 6))
        ttk.Label(row0, text="acars_parser executable:").pack(side="left")
        ttk.Entry(row0, textvariable=self.exe_var, width=80).pack(side="left", padx=6, fill="x", expand=True)
        ttk.Button(row0, text="Browse…", command=self.pick_exe).pack(side="left")

        row1 = ttk.Frame(frm)
        row1.pack(fill="x", pady=6)
        ttk.Label(row1, text="Output folder (optional):").pack(side="left")
        ttk.Entry(row1, textvariable=self.outdir_var, width=80).pack(side="left", padx=6, fill="x", expand=True)
        ttk.Button(row1, text="Browse…", command=self.pick_outdir).pack(side="left")

        row2 = ttk.Frame(frm)
        row2.pack(fill="x", pady=6)
        ttk.Label(row2, text="Output format:").pack(side="left")
        ttk.Radiobutton(row2, text="JSON", value="json", variable=self.format_var, command=self._sync_format_options).pack(side="left", padx=(6, 0))
        ttk.Radiobutton(row2, text="Text", value="text", variable=self.format_var, command=self._sync_format_options).pack(side="left", padx=(6, 0))

        row3 = ttk.Frame(frm)
        row3.pack(fill="x", pady=6)
        self.pretty_check = ttk.Checkbutton(row3, text="Pretty (-pretty)", variable=self.pretty_var)
        self.pretty_check.pack(side="left")
        ttk.Checkbutton(row3, text="All (-all)", variable=self.all_var).pack(side="left", padx=(10, 0))
        ttk.Checkbutton(row3, text="Stats (-stats) if supported", variable=self.stats_var).pack(side="left", padx=(10, 0))
        ttk.Checkbutton(row3, text="Merge outputs into one file", variable=self.merge_var).pack(side="left", padx=(10, 0))
        ttk.Button(row3, text="Add files…", command=self.add_files).pack(side="right")
        ttk.Button(row3, text="Run Extract (queue)", command=self.run_queue).pack(side="right", padx=(0, 8))

        hint = "Drag & drop files into this window to add them to queue." if _HAS_DND else \
               "Drag & drop disabled (install: pip install tkinterdnd2). Use 'Add files…'."
        ttk.Label(frm, text=hint).pack(anchor="w", pady=(4, 2))

        box = ttk.Frame(frm)
        box.pack(fill="both", expand=False, pady=(6, 6))
        ttk.Label(box, text="Queue:").pack(anchor="w")
        self.lst = tk.Listbox(box, height=7, selectmode="extended")
        self.lst.pack(fill="both", expand=True, side="left")
        sb = ttk.Scrollbar(box, orient="vertical", command=self.lst.yview)
        sb.pack(side="left", fill="y")
        self.lst.configure(yscrollcommand=sb.set)

        btncol = ttk.Frame(box)
        btncol.pack(side="left", fill="y", padx=(8, 0))
        ttk.Button(btncol, text="Remove selected", command=self.remove_selected).pack(fill="x", pady=(0, 6))
        ttk.Button(btncol, text="Clear", command=self.clear_queue).pack(fill="x")

        self.txt = tk.Text(frm, wrap="none")
        self.txt.pack(fill="both", expand=True, pady=(10, 0))

        xscroll = ttk.Scrollbar(frm, orient="horizontal", command=self.txt.xview)
        xscroll.pack(fill="x")
        yscroll = ttk.Scrollbar(frm, orient="vertical", command=self.txt.yview)
        yscroll.place(relx=1.0, rely=0.30, relheight=0.64, anchor="ne")
        self.txt.configure(xscrollcommand=xscroll.set, yscrollcommand=yscroll.set)

        self._sync_format_options()

    def pick_exe(self):
        path = filedialog.askopenfilename(
            title="Select acars_parser executable",
            filetypes=[("Executable", "*.exe"), ("All", "*.*")]
        )
        if path:
            self.exe_var.set(path)

    def pick_outdir(self):
        path = filedialog.askdirectory(title="Select output folder")
        if path:
            self.outdir_var.set(path)

    def add_files(self):
        paths = filedialog.askopenfilenames(
            title="Select input files",
            filetypes=[("Logs / JSONL", "*.jsonl;*.log;*.txt;*.json"), ("All", "*.*")]
        )
        if paths:
            self._add_to_queue(list(paths))

    def _on_drop(self, event):
        paths = _split_dnd_files(getattr(event, "data", ""))
        if paths:
            self._add_to_queue(paths)

    def _add_to_queue(self, paths):
        existing = set(self.lst.get(0, "end"))
        for p in paths:
            p = p.strip()
            if not p:
                continue
            if not os.path.exists(p):
                self._append(f"[WARN] Not found: {p}\n")
                continue
            if p in existing:
                continue
            self.lst.insert("end", p)
            existing.add(p)

    def remove_selected(self):
        sel = list(self.lst.curselection())
        sel.reverse()
        for i in sel:
            self.lst.delete(i)

    def clear_queue(self):
        self.lst.delete(0, "end")

    def _output_path_for(self, inp: str) -> str:
        stem, _ext = os.path.splitext(os.path.basename(inp))
        outdir = self.outdir_var.get().strip()
        target_dir = outdir if outdir else os.path.dirname(inp)

        if self.format_var.get() == "json":
            outname = stem + ".json"
        else:
            safe_stem = stem if stem.endswith("+") else stem + "+"
            outname = safe_stem + ".log"

        outpath = os.path.join(target_dir, outname)
        input_path = os.path.normcase(os.path.abspath(inp))

        if os.path.normcase(os.path.abspath(outpath)) == input_path and self.format_var.get() == "text":
            safe_stem = stem + "++"
            outpath = os.path.join(target_dir, safe_stem + ".log")

        return outpath

    def _shared_output_path_for(self, items) -> str:
        first_input = items[0]
        stem, _ext = os.path.splitext(os.path.basename(first_input))
        shared_stem = (stem[:11] or stem or "merged_out").rstrip(" .")
        if not shared_stem:
            shared_stem = "merged_out"

        outdir = self.outdir_var.get().strip()
        target_dir = outdir if outdir else os.path.dirname(first_input)
        suffix = ".json" if self.format_var.get() == "json" else ".log"

        avoid_paths = {
            os.path.normcase(os.path.abspath(path))
            for path in items
        }

        candidate = os.path.join(target_dir, shared_stem + suffix)
        if os.path.normcase(os.path.abspath(candidate)) not in avoid_paths and not os.path.exists(candidate):
            return candidate

        index = 1
        while True:
            candidate = os.path.join(target_dir, f"{shared_stem}_{index}{suffix}")
            if os.path.normcase(os.path.abspath(candidate)) not in avoid_paths and not os.path.exists(candidate):
                return candidate
            index += 1

    def _run_extract_process(self, args, creationflags: int):
        try:
            p = subprocess.Popen(
                args,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                creationflags=creationflags,
            )
            assert p.stdout is not None
            combined = ""
            for line in p.stdout:
                combined += line
                self._append(line)
            rc = p.wait()

            if rc != 0 and self.stats_var.get() and ("flag provided but not defined: -stats" in combined):
                self._append("\n[INFO] This build does not support -stats. Retrying without -stats...\n")
                args = [arg for arg in args if arg != "-stats"]
                self._append("  " + " ".join(args) + "\n")
                p = subprocess.Popen(
                    args,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    text=True,
                    creationflags=creationflags,
                )
                assert p.stdout is not None
                combined = ""
                for line in p.stdout:
                    combined += line
                    self._append(line)
                rc = p.wait()

            return rc, combined
        except Exception as e:
            self._append(f"[ERROR] {e}\n")
            return -1, str(e)

    def _merge_outputs(self, items, exe: str, creationflags: int):
        merged_output = self._shared_output_path_for(items)
        output_format = self.format_var.get().strip().lower() or "json"
        merged_json = []
        merged_text_parts = []
        success_count = 0

        self._append(f"[INFO] Merge mode enabled. Final output: {merged_output}\n")

        for idx, inp in enumerate(items, 1):
            suffix = ".json" if output_format == "json" else ".log"
            temp_file = tempfile.NamedTemporaryFile(delete=False, suffix=suffix)
            temp_file.close()
            temp_path = temp_file.name

            args = [exe, "extract", "-input", inp, "-output", temp_path, "-format", output_format]
            if output_format == "json" and self.pretty_var.get():
                args.append("-pretty")
            if self.all_var.get():
                args.append("-all")
            if self.stats_var.get():
                args.append("-stats")

            self._append(f"\n[{idx}/{len(items)}] Running for merge:\n  " + " ".join(args) + "\n")

            try:
                rc, _combined = self._run_extract_process(args, creationflags)
                if rc != 0 or not os.path.exists(temp_path):
                    self._append(f"[FAIL] Exit code: {rc}\n")
                    self._append(f"       Temporary output: {temp_path}\n")
                    continue

                if output_format == "json":
                    with open(temp_path, "r", encoding="utf-8") as f:
                        payload = json.load(f)
                    if isinstance(payload, list):
                        merged_json.extend(payload)
                    else:
                        merged_json.append(payload)
                else:
                    with open(temp_path, "r", encoding="utf-8") as f:
                        text_payload = f.read()
                    if text_payload:
                        merged_text_parts.append(text_payload.rstrip("\n"))

                success_count += 1
                self._append(f"[OK] Merged input: {inp}\n")
            except Exception as e:
                self._append(f"[ERROR] Failed to merge output from {inp}: {e}\n")
            finally:
                try:
                    os.remove(temp_path)
                except OSError:
                    pass

        if success_count == 0:
            self._append("\n[FAIL] No successful inputs to merge.\n")
            return

        try:
            if output_format == "json":
                with open(merged_output, "w", encoding="utf-8") as f:
                    if self.pretty_var.get():
                        json.dump(merged_json, f, indent=2)
                    else:
                        json.dump(merged_json, f, separators=(",", ":"))
                    f.write("\n")
            else:
                with open(merged_output, "w", encoding="utf-8") as f:
                    content = "\n\n".join(part for part in merged_text_parts if part)
                    if content:
                        f.write(content + "\n")

            self._append(f"\n[OK] Merged output written: {merged_output}\n")
        except Exception as e:
            self._append(f"\n[ERROR] Failed to write merged output: {e}\n")

    def _sync_format_options(self):
        is_json = self.format_var.get() == "json"
        if not is_json:
            self.pretty_var.set(False)
        self.pretty_check.configure(state="normal" if is_json else "disabled")

    def run_queue(self):
        exe = self.exe_var.get().strip()
        if not exe:
            messagebox.showerror("Missing", "Select acars_parser executable.")
            return
        if not os.path.exists(exe):
            messagebox.showerror("Not found", f"Executable not found:\n{exe}")
            return

        items = list(self.lst.get(0, "end"))
        if not items:
            messagebox.showinfo("Queue empty", "Add one or more input files first.")
            return

        self.txt.delete("1.0", "end")
        self._append(f"Queue length: {len(items)}\n\n")

        def worker():
            creationflags = 0
            if os.name == "nt":
                creationflags = getattr(subprocess, "CREATE_NO_WINDOW", 0)

            if self.merge_var.get():
                self._merge_outputs(items, exe, creationflags)
                self._append("\nDone.\n")
                return

            for idx, inp in enumerate(items, 1):
                outp = self._output_path_for(inp)
                output_format = self.format_var.get().strip().lower() or "json"

                args = [exe, "extract", "-input", inp, "-output", outp, "-format", output_format]
                if output_format == "json" and self.pretty_var.get():
                    args.append("-pretty")
                if self.all_var.get():
                    args.append("-all")
                if self.stats_var.get():
                    args.append("-stats")

                self._append(f"\n[{idx}/{len(items)}] Running:\n  " + " ".join(args) + "\n")
                rc, _combined = self._run_extract_process(args, creationflags)

                if rc == 0 and os.path.exists(outp):
                    self._append(f"[OK] Output: {outp}\n")
                else:
                    self._append(f"[FAIL] Exit code: {rc}\n")
                    self._append(f"       Expected output: {outp}\n")

            self._append("\nDone.\n")

        threading.Thread(target=worker, daemon=True).start()

    def _append(self, s: str):
        def _do():
            self.txt.insert("end", s)
            self.txt.see("end")
        self.after(0, _do)


if __name__ == "__main__":
    App().mainloop()
