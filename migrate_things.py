#!/usr/bin/env python3
"""Migrate Things 3 database to atask.

Migrates all non-trashed tasks created in 2025-2026, along with their
areas, tags, projects, sections, and checklist items.
"""

import json
import sqlite3
import sys
import time
import urllib.request
import urllib.error

THINGS_DB = "Things Database.thingsdatabase/main.sqlite"
ATASK_BASE = "http://localhost:8080"
COCOA_2025 = 757382400.0  # 2025-01-01 as Cocoa timestamp (seconds since 2001-01-01)


def decode_things_date(val):
    """Decode Things 3 integer date to YYYY-MM-DD string."""
    if val is None:
        return None
    year = val >> 16
    remainder = val & 0xFFFF
    month = (remainder >> 12) & 0xF
    day = (remainder >> 7) & 0x1F
    if year < 2000 or month < 1 or month > 12 or day < 1 or day > 31:
        return None
    return f"{year}-{month:02d}-{day:02d}"


def api(method, path, token, body=None):
    """Make an HTTP request to the atask API."""
    url = f"{ATASK_BASE}{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", f"Bearer {token}")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        err_body = e.read().decode()
        print(f"  ERROR {e.code} {method} {path}: {err_body}")
        return None


def get_token():
    """Login or register and return JWT token."""
    body = json.dumps({"email": "arthur@example.com", "password": "migrate123"}).encode()

    # Try login first
    req = urllib.request.Request(f"{ATASK_BASE}/auth/login", data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())["token"]
    except urllib.error.HTTPError:
        pass

    # Register
    reg_body = json.dumps({"email": "arthur@example.com", "password": "migrate123", "name": "Arthur"}).encode()
    req = urllib.request.Request(f"{ATASK_BASE}/auth/register", data=reg_body, method="POST")
    req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req) as resp:
        json.loads(resp.read())

    # Login
    req = urllib.request.Request(f"{ATASK_BASE}/auth/login", data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read())["token"]


def main():
    conn = sqlite3.connect(THINGS_DB)
    conn.row_factory = sqlite3.Row
    token = get_token()
    print(f"Authenticated.\n")

    # Maps: Things UUID -> atask ID
    area_map = {}
    tag_map = {}
    project_map = {}
    section_map = {}  # heading UUID -> atask section ID
    task_map = {}

    # ── 1. Collect Things UUIDs of projects referenced by 2025+ tasks ──
    referenced_projects = set()
    rows = conn.execute("""
        SELECT DISTINCT project FROM TMTask
        WHERE type=0 AND trashed=0 AND creationDate >= ?
        AND project IS NOT NULL AND project != ''
    """, (COCOA_2025,)).fetchall()
    for r in rows:
        referenced_projects.add(r["project"])

    # Also include open projects (status=0, not trashed) even if no 2025+ tasks yet
    rows = conn.execute("SELECT uuid FROM TMTask WHERE type=1 AND status=0 AND trashed=0").fetchall()
    for r in rows:
        referenced_projects.add(r["uuid"])

    # ── 2. Areas ──
    areas = conn.execute("SELECT uuid, title FROM TMArea ORDER BY \"index\"").fetchall()
    print(f"Creating {len(areas)} areas...")
    for a in areas:
        title = a["title"] or "Untitled Area"
        resp = api("POST", "/areas", token, {"title": title})
        if resp:
            area_map[a["uuid"]] = resp["data"]["ID"]
            print(f"  Area: {title}")

    # ── 3. Tags (skip parent tags, create flat) ──
    tags = conn.execute("SELECT uuid, title FROM TMTag ORDER BY \"index\"").fetchall()
    print(f"\nCreating {len(tags)} tags...")
    for t in tags:
        title = t["title"] or "Untitled Tag"
        resp = api("POST", "/tags", token, {"title": title})
        if resp:
            tag_map[t["uuid"]] = resp["data"]["ID"]
            print(f"  Tag: {title}")

    # ── 4. Projects (only referenced ones) ──
    projects = conn.execute("""
        SELECT uuid, title, area, status, notes FROM TMTask
        WHERE type=1 AND trashed=0 AND uuid IN ({})
        ORDER BY "index"
    """.format(",".join("?" for _ in referenced_projects)), list(referenced_projects)).fetchall()
    print(f"\nCreating {len(projects)} projects...")
    for p in projects:
        title = p["title"] or "Untitled Project"
        resp = api("POST", "/projects", token, {"title": title})
        if not resp:
            continue
        pid = resp["data"]["ID"]
        project_map[p["uuid"]] = pid
        print(f"  Project: {title}")

        # Set notes
        if p["notes"]:
            api("PUT", f"/projects/{pid}/notes", token, {"notes": p["notes"]})

        # Set area
        if p["area"] and p["area"] in area_map:
            api("PUT", f"/projects/{pid}/area", token, {"id": area_map[p["area"]]})

        # Set status
        if p["status"] == 3:
            api("POST", f"/projects/{pid}/complete", token)
        elif p["status"] == 2:
            api("POST", f"/projects/{pid}/cancel", token)

    # ── 5. Sections (headings in referenced projects) ──
    headings = conn.execute("""
        SELECT uuid, title, project FROM TMTask
        WHERE type=2 AND trashed=0 AND project IN ({})
        ORDER BY "index"
    """.format(",".join("?" for _ in referenced_projects)), list(referenced_projects)).fetchall()
    print(f"\nCreating {len(headings)} sections...")
    for h in headings:
        if h["project"] not in project_map:
            continue
        pid = project_map[h["project"]]
        title = h["title"] or "Untitled Section"
        resp = api("POST", f"/projects/{pid}/sections", token, {"title": title})
        if resp:
            section_map[h["uuid"]] = resp["data"]["ID"]
            print(f"  Section: {title}")

    # ── 6. Tasks (created in 2025+, not trashed) ──
    tasks = conn.execute("""
        SELECT uuid, title, notes, start, startDate, deadline,
               project, heading, area, status
        FROM TMTask
        WHERE type=0 AND trashed=0 AND creationDate >= ?
        ORDER BY "index"
    """, (COCOA_2025,)).fetchall()
    print(f"\nCreating {len(tasks)} tasks...")
    created = 0
    errors = 0
    for t in tasks:
        title = t["title"] or "Untitled Task"
        resp = api("POST", "/tasks", token, {"title": title})
        if not resp:
            errors += 1
            continue
        tid = resp["data"]["ID"]
        task_map[t["uuid"]] = tid
        created += 1

        # Notes
        if t["notes"]:
            api("PUT", f"/tasks/{tid}/notes", token, {"notes": t["notes"]})

        # Schedule: Things start 0=inbox(not started), 1=anytime, 2=someday
        schedule_map = {0: "inbox", 1: "anytime", 2: "someday"}
        schedule = schedule_map.get(t["start"], "anytime")
        api("PUT", f"/tasks/{tid}/schedule", token, {"schedule": schedule})

        # Start date
        start_date = decode_things_date(t["startDate"])
        if start_date:
            api("PUT", f"/tasks/{tid}/start-date", token, {"date": start_date})

        # Deadline
        deadline = decode_things_date(t["deadline"])
        if deadline:
            api("PUT", f"/tasks/{tid}/deadline", token, {"date": deadline})

        # Project
        if t["project"] and t["project"] in project_map:
            api("PUT", f"/tasks/{tid}/project", token, {"id": project_map[t["project"]]})

        # Section (heading)
        if t["heading"] and t["heading"] in section_map:
            api("PUT", f"/tasks/{tid}/section", token, {"id": section_map[t["heading"]]})

        # Area (only if no project — atask may handle this differently)
        if t["area"] and t["area"] in area_map and not t["project"]:
            api("PUT", f"/tasks/{tid}/area", token, {"id": area_map[t["area"]]})

        # Status
        if t["status"] == 3:
            api("POST", f"/tasks/{tid}/complete", token)
        elif t["status"] == 2:
            api("POST", f"/tasks/{tid}/cancel", token)

        if created % 100 == 0:
            print(f"  ... {created} tasks created")

    print(f"  Done: {created} tasks created, {errors} errors")

    # ── 7. Checklist items ──
    checklists = conn.execute("""
        SELECT ci.uuid, ci.title, ci.status, ci.task
        FROM TMChecklistItem ci
        JOIN TMTask t ON ci.task = t.uuid
        WHERE t.type=0 AND t.trashed=0 AND t.creationDate >= ?
        ORDER BY ci."index"
    """, (COCOA_2025,)).fetchall()
    print(f"\nCreating {len(checklists)} checklist items...")
    for ci in checklists:
        if ci["task"] not in task_map:
            continue
        tid = task_map[ci["task"]]
        title = ci["title"] or "Untitled Item"
        resp = api("POST", f"/tasks/{tid}/checklist", token, {"title": title})
        if resp and ci["status"] == 3:
            item_id = resp["data"]["ID"]
            api("POST", f"/tasks/{tid}/checklist/{item_id}/complete", token)
        if resp:
            print(f"  Checklist: {title}")

    # ── 8. Task-tag associations ──
    task_tags = conn.execute("""
        SELECT tt.tasks, tt.tags
        FROM TMTaskTag tt
        JOIN TMTask t ON tt.tasks = t.uuid
        WHERE t.type=0 AND t.trashed=0 AND t.creationDate >= ?
    """, (COCOA_2025,)).fetchall()
    print(f"\nAssigning {len(task_tags)} task-tag links...")
    for tt in task_tags:
        if tt["tasks"] in task_map and tt["tags"] in tag_map:
            tid = task_map[tt["tasks"]]
            tag_id = tag_map[tt["tags"]]
            api("POST", f"/tasks/{tid}/tags/{tag_id}", token)

    # ── 9. Project-tag associations ──
    # Things uses TMAreaTag for project tags too (confusingly)
    proj_tags = conn.execute("""
        SELECT tt.tasks, tt.tags
        FROM TMTaskTag tt
        JOIN TMTask t ON tt.tasks = t.uuid
        WHERE t.type=1 AND t.trashed=0 AND t.uuid IN ({})
    """.format(",".join("?" for _ in referenced_projects)), list(referenced_projects)).fetchall()
    print(f"Assigning {len(proj_tags)} project-tag links...")
    for pt in proj_tags:
        if pt["tasks"] in project_map and pt["tags"] in tag_map:
            pid = project_map[pt["tasks"]]
            tag_id = tag_map[pt["tags"]]
            api("POST", f"/projects/{pid}/tags/{tag_id}", token)

    print(f"\n{'='*50}")
    print(f"Migration complete!")
    print(f"  Areas:     {len(area_map)}")
    print(f"  Tags:      {len(tag_map)}")
    print(f"  Projects:  {len(project_map)}")
    print(f"  Sections:  {len(section_map)}")
    print(f"  Tasks:     {len(task_map)}")
    print(f"  Checklists: {len(checklists)}")
    print(f"  Task-tags: {len(task_tags)}")


if __name__ == "__main__":
    main()
