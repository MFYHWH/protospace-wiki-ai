# Forum Post Options

## Version A

### Making the Wiki Work with AI

I actually found Protospace through Claude — I was asking about makerspaces in Calgary and it pointed me here. But when I tried asking follow-up questions about the space, it couldn't tell me much. The wiki has a ton of great info but AI tools can't see any of it right now.

I think that's worth fixing for two reasons. First, more people could find Protospace the way I did — through AI search. Right now if someone asks Claude or ChatGPT about makerspaces in Calgary, Protospace barely comes up because the wiki is invisible to them. Second, members could actually use AI to search the wiki. Things like "what are the safety rules for the laser cutter" or "how do I book the CNC" — instead of digging through pages manually.

I put together a small server-side change that would open up the wiki to AI crawlers. It's a one-time setup, nothing changes for members, and it's fully reversible if anyone has concerns. Repo is here if anyone wants to take a look: https://github.com/MFYHWH/protospace-wiki-ai

Happy to walk through it or answer questions.

---

## Version B

### Proposal: Let AI read the wiki

Quick backstory — Claude actually introduced me to Protospace. I was chatting with it about makerspaces in Calgary and it mentioned this place. But when I started asking it specific questions about the space, it drew a blank. Turns out none of the AI tools (Claude, ChatGPT, Gemini, Perplexity) can actually read the wiki.

I'd love to change that. The upside is twofold — it helps new people discover Protospace through AI the same way I did, and it means existing members can just ask their AI about the wiki instead of hunting through it manually. Stuff like safety procedures, equipment info, how-tos — it's all already written up, AI just can't get to it.

I've put together a lightweight fix that runs on the server. One-time deploy, no impact on members, easy to undo. Details and code here: https://github.com/MFYHWH/protospace-wiki-ai

Let me know what you think — happy to discuss.
