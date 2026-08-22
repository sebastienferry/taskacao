import React from 'react'

interface TaskacaoLogoProps {
  size?: number
  className?: string
  withBadge?: boolean
}

/**
 * TaskacaoLogo - Official logo representing a warm, steaming cup of hot chocolate ("tasse d'un bon chocolat")
 * Designed for crisp vector scaling and high contrast in both Dark and Light themes.
 */
export const TaskacaoLogo: React.FC<TaskacaoLogoProps> = ({
  size = 32,
  className = '',
  withBadge = true,
}) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 36 36"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-label="Taskacao Logo"
    >
      <defs>
        {/* Rich Cocoa Badge Gradient */}
        <linearGradient id="taskacao-badge-grad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#78350f" />
          <stop offset="50%" stopColor="#451a03" />
          <stop offset="100%" stopColor="#290f02" />
        </linearGradient>

        {/* Cup Ceramic Gradient - Warm Golden Amber */}
        <linearGradient id="taskacao-cup-grad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#fef3c7" />
          <stop offset="60%" stopColor="#fed7aa" />
          <stop offset="100%" stopColor="#f97316" />
        </linearGradient>

        {/* Hot Chocolate Liquid Gradient */}
        <linearGradient id="taskacao-choco-grad" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor="#3b1303" />
          <stop offset="100%" stopColor="#1a0701" />
        </linearGradient>

        {/* Cream / Marshmallow Foam Gradient */}
        <linearGradient id="taskacao-foam-grad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#ffffff" />
          <stop offset="100%" stopColor="#fde68a" />
        </linearGradient>

        {/* Steam Plumes Gradient */}
        <linearGradient id="taskacao-steam-grad" x1="0%" y1="100%" x2="0%" y2="0%">
          <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.9" />
          <stop offset="50%" stopColor="#fbbf24" stopOpacity="0.6" />
          <stop offset="100%" stopColor="#fef3c7" stopOpacity="0" />
        </linearGradient>
      </defs>

      {/* Optional Cocoa Badge Container */}
      {withBadge && (
        <>
          <rect
            width="36"
            height="36"
            rx="9"
            fill="url(#taskacao-badge-grad)"
          />
          <rect
            x="0.5"
            y="0.5"
            width="35"
            height="35"
            rx="8.5"
            stroke="#d97706"
            strokeOpacity="0.35"
            strokeWidth="1"
          />
        </>
      )}

      {/* Steam Trails */}
      <path
        d="M12 13c-0.9-2.2 0.7-3.8 0-6"
        stroke="url(#taskacao-steam-grad)"
        strokeWidth="1.6"
        strokeLinecap="round"
        fill="none"
      />
      <path
        d="M15.5 12c-1-2.5 0.9-4.5 0-7"
        stroke="url(#taskacao-steam-grad)"
        strokeWidth="1.8"
        strokeLinecap="round"
        fill="none"
      />
      <path
        d="M19 13c-0.9-2.2 0.7-3.8 0-6"
        stroke="url(#taskacao-steam-grad)"
        strokeWidth="1.6"
        strokeLinecap="round"
        fill="none"
      />

      {/* Cup Handle */}
      <path
        d="M21.5 16.5h2.2c1.7 0 2.8 1.1 2.8 2.5v0.6c0 1.4-1.1 2.5-2.8 2.5h-2.7"
        stroke="url(#taskacao-cup-grad)"
        strokeWidth="2"
        strokeLinecap="round"
        fill="none"
      />

      {/* Cup Base / Saucer hint */}
      <ellipse
        cx="15"
        cy="28.5"
        rx="7.5"
        ry="1.5"
        fill="#451a03"
        fillOpacity="0.4"
      />

      {/* Cup Body */}
      <path
        d="M8.5 15.5h13c0 0-0.4 7.2-2.2 9.5-1.3 1.6-2.7 1.8-4.3 1.8s-3-0.2-4.3-1.8C8.9 22.7 8.5 15.5 8.5 15.5Z"
        fill="url(#taskacao-cup-grad)"
      />

      {/* Cup Outer Rim */}
      <ellipse
        cx="15"
        cy="15.5"
        rx="6.5"
        ry="2.3"
        fill="#fef3c7"
      />

      {/* Hot Chocolate Surface */}
      <ellipse
        cx="15"
        cy="15.7"
        rx="5.5"
        ry="1.7"
        fill="url(#taskacao-choco-grad)"
      />

      {/* Cream / Latte Art Chocolate Swirl Heart */}
      <path
        d="M13.8 15.1c-0.5-0.5-1.3 0.1-0.8 0.7 0.6 0.7 2 1.3 2 1.3s1.4-0.6 2-1.3c0.5-0.6-0.3-1.2-0.8-0.7-0.5 0.5-1.2 0.9-1.2 0.9s-0.7-0.4-1.2-0.9Z"
        fill="url(#taskacao-foam-grad)"
      />
      <circle cx="12.2" cy="15.6" r="0.5" fill="#fde68a" />
      <circle cx="17.6" cy="15.7" r="0.45" fill="#fde68a" />
    </svg>
  )
}
