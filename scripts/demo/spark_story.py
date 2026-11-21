"""The Spark story table: who, what and in which state, with no I/O.

seed_dating_scenario.py drives the API from these tables; the unittest in
scripts/tests/test_spark_story.py checks they are well formed. Every address is
on a reserved .test name, never a real mailbox.
"""

DOMAIN = "spark.example.test"
WORKSPACE = "Spark Labs"
PROJECT_ROOT = "Spark MVP"

# key, display name, title, workspace role at the end of the story
PEOPLE = [
    ("maya", "Maya Chen", "Product lead", "admin"),
    ("leo", "Leo Park", "Engineering lead", "member"),
    ("priya", "Priya Nair", "Product designer", "member"),
    ("sam", "Sam Okafor", "Backend engineer", "member"),
    ("jo", "Jo Rivera", "QA and trust & safety", "admin"),  # promoted by Maya
]
CREATOR = "maya"
PROMOTED = "jo"


def email(key):
    return f"{key}@{DOMAIN}"


# author, title, content; the first idea is the one converted to the PRD
IDEAS = [
    ("maya", "Spark: dating for slow, intentional matching",
     "A dating app that sends a few thoughtful matches a day, built on profile "
     "prompts instead of endless swiping."),
    ("maya", "Voice-note prompts",
     "Let members answer one prompt with a 20-second voice note."),
    ("leo", "Match-quality offline eval", "Replay last week's matches against a new scorer."),
    ("priya", "Calm mode theme", "A low-contrast evening theme that slows the pace of the app."),
    ("sam", "Chat retention job", "Expire chats 90 days after an unmatch."),
    ("jo", "Trusted-contact date check-in", "Share date plans with a friend who gets a check-in ping."),
]

# PRD sections (store keys); "background" carries the problem statement
PRD_TITLE = "Spark: slow, intentional matching"
PRD_SECTIONS = {
    "background": "Problem: swipe apps optimise for volume. Members feel burned out, "
                  "conversations stall, and safety reports pile up.",
    "users": "Adults 25-40 in two launch cities who want a relationship, have tried "
             "swipe apps and left them tired.",
    "goals": "1. Half of matches reach a five-message conversation. "
             "2. Members return weekly for their daily matches. "
             "3. Every report is reviewed within 24 hours.",
    "nonGoals": "No swiping deck, no paid boosts, no public popularity scores, "
                "no video calls in v1.",
    "requirements": "- Profiles: photos plus three written prompts.\n"
                    "- Prompts-based matching: scores compatibility from prompt answers.\n"
                    "- Daily match limit: at most three new matches a day.\n"
                    "- Chat: one-to-one messaging, unmatch ends the chat.\n"
                    "- Safety: report and block from every profile and chat.\n"
                    "- Verification: selfie check grants a verified badge.",
    "stories": "As a member I open the app once a day, read three profiles, and start "
               "one conversation from a prompt.",
    "constraints": "Two engineers and one designer; closed beta in eight weeks.",
    "risks": "Too few members per city for three good daily matches; moderators "
             "overwhelmed at launch.",
}
COACH_MESSAGE = ("Our core user is someone tired of swipe apps. Is a daily limit of three "
                 "matches the right first bet?")
REVISION_NOTES = ["First full draft of all sections", "Tightened goals after the coach turn"]
COMMIT = {
    "name": "Spark v1 baseline",
    "primaryAudience": "decision",
    "changeSummary": "First approved scope: prompts matching, daily limit, chat, safety.",
    "unresolved": ["Minimum members per city for three daily matches"],
    "decisionNote": "Approved by trust & safety; launch scope frozen for the beta.",
}

# Roadmap branches under PROJECT_ROOT. A node is a dict:
#   title, owner (delegate to; a branch owner delegates below it), status
#   (pending | in_progress | done | validated), due (days from today, may be
#   negative = past due), acceptance, evidence ([(detail, url-or-"")]),
#   handback (owner returns it to whoever delegated it), children.
TREE = [
    {"title": "Onboarding & profiles", "owner": "priya", "children": [
        {"title": "Sign-up and phone verification", "status": "done", "due": -6,
         "acceptance": "A new member signs up and verifies a phone number in under two minutes.",
         "evidence": [("Usability test: 5 of 5 finished sign-up unaided", "https://spark.example.test/research/signup")]},
        {"title": "Profile builder", "children": [
            {"title": "Photo upload with crop", "status": "in_progress", "due": 3,
             "acceptance": "Up to six photos, each cropped to 4:5 before upload."},
            {"title": "Prompt picker", "status": "pending", "due": 12,
             "acceptance": "A member answers three of twenty prompts, 200 characters each."},
        ]},
        {"title": "Selfie verification badge", "status": "pending", "due": 30,
         "acceptance": "A verified badge shows after a passed selfie check."},
        {"title": "Onboarding copy and tone review", "status": "done", "due": -2,
         "acceptance": "All onboarding screens use the calm voice guide.",
         "evidence": [("Copy reviewed with the voice guide; no open comments", "")]},
    ]},
    {"title": "Matching", "owner": "leo", "children": [
        {"title": "Prompt-based matching engine", "owner": "sam", "children": [
            {"title": "Prompt answer embeddings", "status": "done", "due": -4,
             "acceptance": "Every prompt answer has an embedding within a minute of saving.",
             "evidence": [("Backfill job ran on staging; 100% of answers embedded", "https://spark.example.test/ci/embeddings")]},
            {"title": "Compatibility scoring", "status": "in_progress", "due": 4,
             "acceptance": "Scores rank a hand-labelled set with precision at 3 of 0.6 or better."},
            {"title": "Match explanation copy", "status": "pending", "due": 18,
             "acceptance": "Each match shows one sentence on the prompt it was matched on."},
        ]},
        {"title": "Daily match limit", "owner": "sam", "status": "in_progress", "due": 2,
         "acceptance": "No member receives more than three new matches in a calendar day."},
        {"title": "Daily drop UI", "status": "pending", "due": 9,
         "acceptance": "Matches appear together at 7 pm local time with a calm reveal."},
        {"title": "Matching quality dashboard", "status": "pending", "due": 35,
         "acceptance": "Weekly view of five-message conversation rate per city."},
    ]},
    {"title": "Chat", "owner": "leo", "children": [
        {"title": "Realtime messaging service", "owner": "sam", "status": "in_progress", "due": -3,
         "acceptance": "Messages arrive in under one second for 99% of sends."},
        {"title": "Conversation starters from prompts", "status": "pending", "due": 10,
         "acceptance": "A chat opens with a suggested reply to the other member's prompt."},
        {"title": "Unmatch ends chat", "status": "done", "due": -1,
         "acceptance": "After an unmatch neither member can send or read the chat.",
         "evidence": [("Integration test passes on staging", "https://spark.example.test/ci/unmatch")]},
    ]},
    {"title": "Safety & trust", "owner": "jo", "children": [
        {"title": "Report a member", "children": [
            {"title": "Report flow UI", "status": "done", "due": -5,
             "acceptance": "Report is two taps from any profile or chat.",
             "evidence": [("QA pass on iOS and Android builds", "https://spark.example.test/qa/report-flow")]},
            {"title": "Moderator review queue", "status": "in_progress", "due": -1,
             "acceptance": "Every report reaches a moderator within 24 hours."},
        ]},
        {"title": "Block and hide", "status": "validated", "due": -3,
         "acceptance": "A blocked member never appears again, in matches or chat.",
         "evidence": [("Block test matrix passes: matches, chat and search", "https://spark.example.test/qa/block"),
                      ("Beta tester confirmed the blocked profile never came back", "")],
         "validation": "Beta testers in both cities confirmed blocked members stayed hidden for a week."},
        {"title": "Photo verification checks", "status": "pending", "due": 28,
         "acceptance": "Profile photos are checked against the verification selfie."},
        {"title": "Safety center page", "status": "pending", "due": 21,
         "acceptance": "One page explains reporting, blocking and date safety tips."},
    ]},
    {"title": "Launch", "children": [
        {"title": "Closed beta in two cities", "status": "pending", "due": 40,
         "acceptance": "200 members per city before matching is switched on."},
        {"title": "App Store listing and screenshots", "owner": "priya", "handback": True,
         "status": "pending", "due": 25,
         "acceptance": "Listing approved in both stores with six screenshots."},
        {"title": "Launch metrics and success criteria", "status": "in_progress", "due": 5,
         "acceptance": "Success criteria agreed and wired to the dashboard."},
        {"title": "Privacy policy and terms review", "status": "pending", "due": -2,
         "acceptance": "Counsel signs off on the privacy policy and terms."},
    ]},
]

# Proposals: (author, node it is about, kind, summary, item titles, parentHint, decision)
PROPOSALS = [
    ("sam", "Prompt-based matching engine", "structure",
     "Add a cold-start fallback so new members still get matches before their answers embed.",
     ["Cold-start fallback for new members"], "Prompt-based matching engine", "accepted"),
    ("jo", "Photo verification checks", "scope",
     "Add a third-party liveness vendor evaluation before photo checks.",
     ["Liveness vendor evaluation"], "Photo verification checks", "rejected"),
]

# Project updates: (author, kind, content)
UPDATES = [
    ("maya", "note", "Spark v1 baseline committed; the roadmap is built from it."),
    ("leo", "progress", "Embeddings backfill finished on staging."),
    ("priya", "progress", "Profile builder prototype tested with five people."),
    ("sam", "blocker", "Realtime messaging needs a managed queue; waiting on a budget decision."),
    ("jo", "progress", "Report flow passed QA on both platforms."),
    ("maya", "note", "Budget for the managed queue approved; Sam is unblocked."),
    ("sam", "progress", "Daily limit enforced in the match job; tests green."),
    ("jo", "blocker", "Moderator queue overdue: we need two more moderators for launch week."),
    ("priya", "note", "Handed the App Store listing back to Maya; marketing will own it."),
    ("leo", "progress", "Unmatch now closes the chat for both members."),
]
STATUSES = ("pending", "in_progress", "done", "validated")


def walk(nodes=None, parent=None):
    """Yield (parent title or None, node) depth first."""
    for n in TREE if nodes is None else nodes:
        yield parent, n
        yield from walk(n.get("children", []), n["title"])
