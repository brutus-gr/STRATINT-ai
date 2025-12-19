import { useState } from 'react';
import { Share2, Twitter, Linkedin, Mail, Link as LinkIcon, Check } from 'lucide-react';
import type { Event } from '../types';

interface ShareButtonsProps {
  event: Event;
}

export function ShareButtons({ event }: ShareButtonsProps) {
  const [copied, setCopied] = useState(false);

  // Get current page URL
  const eventUrl = window.location.href;

  // Create share text for different platforms
  const getShareText = (platform: 'twitter' | 'linkedin' | 'email') => {
    const baseText = `🚨 ${event.title}`;
    const category = `#${event.category.toUpperCase()}`;
    const magnitude = event.magnitude >= 7 ? '🔴' : event.magnitude >= 5 ? '🟡' : '';

    switch (platform) {
      case 'twitter':
        // Twitter has 280 char limit
        return `${magnitude} ${baseText}\n\nMagnitude: ${event.magnitude.toFixed(1)}/10 | Confidence: ${(event.confidence.score * 100).toFixed(0)}%\n\n${category} #OSINT`;
      case 'linkedin':
        return `${baseText}\n\nMagnitude: ${event.magnitude.toFixed(1)}/10\nConfidence: ${(event.confidence.score * 100).toFixed(0)}%\nCategory: ${event.category}\n\nSource: OSINT Intelligence Platform`;
      case 'email':
        return baseText;
      default:
        return baseText;
    }
  };

  const getEmailBody = () => {
    return `${event.title}

Magnitude: ${event.magnitude.toFixed(1)}/10
Confidence: ${(event.confidence.score * 100).toFixed(0)}%
Category: ${event.category}

${event.confidence.reasoning || ''}

View full event: ${eventUrl}

---
Shared from OSINT Intelligence Platform`;
  };

  const handleTwitterShare = () => {
    const text = encodeURIComponent(getShareText('twitter'));
    const url = encodeURIComponent(eventUrl);
    window.open(`https://twitter.com/intent/tweet?text=${text}&url=${url}`, '_blank', 'width=550,height=420');
  };

  const handleLinkedInShare = () => {
    const url = encodeURIComponent(eventUrl);
    window.open(`https://www.linkedin.com/sharing/share-offsite/?url=${url}`, '_blank', 'width=550,height=420');
  };

  const handleRedditShare = () => {
    const title = encodeURIComponent(event.title);
    const url = encodeURIComponent(eventUrl);
    window.open(`https://reddit.com/submit?title=${title}&url=${url}`, '_blank', 'width=550,height=420');
  };

  const handleEmailShare = () => {
    const subject = encodeURIComponent(`[OSINT] ${event.title}`);
    const body = encodeURIComponent(getEmailBody());
    window.location.href = `mailto:?subject=${subject}&body=${body}`;
  };

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(eventUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy link:', err);
    }
  };

  return (
    <div className="border-2 border-steel bg-void/30 p-4">
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div className="flex items-center gap-2">
          <Share2 className="w-4 h-4 text-terminal" />
          <span className="font-mono text-sm font-bold text-chalk">SHARE EVENT</span>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {/* Twitter */}
          <button
            onClick={handleTwitterShare}
            className="group flex items-center gap-2 px-3 py-2 border-2 border-steel bg-void hover:border-terminal hover:bg-terminal/10 transition-all"
            title="Share on Twitter / X"
          >
            <Twitter className="w-4 h-4 text-chalk group-hover:text-terminal transition-colors" />
            <span className="font-mono text-xs font-bold text-chalk group-hover:text-terminal transition-colors hidden sm:inline">
              X
            </span>
          </button>

          {/* LinkedIn */}
          <button
            onClick={handleLinkedInShare}
            className="group flex items-center gap-2 px-3 py-2 border-2 border-steel bg-void hover:border-electric hover:bg-electric/10 transition-all"
            title="Share on LinkedIn"
          >
            <Linkedin className="w-4 h-4 text-chalk group-hover:text-electric transition-colors" />
            <span className="font-mono text-xs font-bold text-chalk group-hover:text-electric transition-colors hidden sm:inline">
              LINKEDIN
            </span>
          </button>

          {/* Reddit */}
          <button
            onClick={handleRedditShare}
            className="group flex items-center gap-2 px-3 py-2 border-2 border-steel bg-void hover:border-threat-medium hover:bg-threat-medium/10 transition-all"
            title="Share on Reddit"
          >
            <Share2 className="w-4 h-4 text-chalk group-hover:text-threat-medium transition-colors" />
            <span className="font-mono text-xs font-bold text-chalk group-hover:text-threat-medium transition-colors hidden sm:inline">
              REDDIT
            </span>
          </button>

          {/* Email */}
          <button
            onClick={handleEmailShare}
            className="group flex items-center gap-2 px-3 py-2 border-2 border-steel bg-void hover:border-cyber hover:bg-cyber/10 transition-all"
            title="Share via Email"
          >
            <Mail className="w-4 h-4 text-chalk group-hover:text-cyber transition-colors" />
            <span className="font-mono text-xs font-bold text-chalk group-hover:text-cyber transition-colors hidden sm:inline">
              EMAIL
            </span>
          </button>

          {/* Copy Link */}
          <button
            onClick={handleCopyLink}
            className={`group flex items-center gap-2 px-3 py-2 border-2 transition-all ${
              copied
                ? 'border-terminal bg-terminal/10'
                : 'border-steel bg-void hover:border-terminal hover:bg-terminal/10'
            }`}
            title="Copy Link"
          >
            {copied ? (
              <>
                <Check className="w-4 h-4 text-terminal" />
                <span className="font-mono text-xs font-bold text-terminal hidden sm:inline">
                  COPIED!
                </span>
              </>
            ) : (
              <>
                <LinkIcon className="w-4 h-4 text-chalk group-hover:text-terminal transition-colors" />
                <span className="font-mono text-xs font-bold text-chalk group-hover:text-terminal transition-colors hidden sm:inline">
                  COPY LINK
                </span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
