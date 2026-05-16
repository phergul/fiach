import { useCallback, useLayoutEffect, useRef } from 'react';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { GameCard } from '@components/Games/Grid/GameCard/GameCard';

import './GameGrid.scss';

interface GameGridProps {
  games: StoredGame[];
}

interface GameGridItemPosition {
  left: number;
  top: number;
}

const gameGridMotionDuration = 160;
const gameGridMotionDistanceThreshold = 0.5;

const prefersReducedMotion = () => {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
};

export const GameGrid = ({ games }: GameGridProps) => {
  const gridRef = useRef<HTMLDivElement | null>(null);
  const cardRefs = useRef(new Map<number, HTMLElement>());
  const previousPositions = useRef(new Map<number, GameGridItemPosition>());
  const activeAnimations = useRef(new Map<number, Animation>());
  const animationFrame = useRef<number | null>(null);

  const measureCards = useCallback(() => {
    const nextPositions = new Map<number, GameGridItemPosition>();

    cardRefs.current.forEach((cardElement, gameID) => {
      nextPositions.set(gameID, {
        left: cardElement.offsetLeft,
        top: cardElement.offsetTop,
      });
    });

    return nextPositions;
  }, []);

  const cancelActiveAnimations = useCallback(() => {
    activeAnimations.current.forEach((animation) => {
      animation.cancel();
    });

    activeAnimations.current.clear();
  }, []);

  const runLayoutAnimation = useCallback(() => {
    const nextPositions = measureCards();

    if (prefersReducedMotion()) {
      cancelActiveAnimations();
      previousPositions.current = nextPositions;
      return;
    }

    cardRefs.current.forEach((cardElement, gameID) => {
      const previousPosition = previousPositions.current.get(gameID);
      const nextPosition = nextPositions.get(gameID);

      if (previousPosition === undefined || nextPosition === undefined) {
        return;
      }

      const deltaX = previousPosition.left - nextPosition.left;
      const deltaY = previousPosition.top - nextPosition.top;
      const hasMoved =
        Math.abs(deltaX) > gameGridMotionDistanceThreshold ||
        Math.abs(deltaY) > gameGridMotionDistanceThreshold;

      if (!hasMoved) {
        return;
      }

      activeAnimations.current.get(gameID)?.cancel();

      const animation = cardElement.animate([
        { transform: `translate(${deltaX}px, ${deltaY}px)` },
        { transform: 'translate(0, 0)' },
      ], {
        duration: gameGridMotionDuration,
        easing: 'ease',
      });

      activeAnimations.current.set(gameID, animation);

      animation.addEventListener('finish', () => {
        if (activeAnimations.current.get(gameID) === animation) {
          activeAnimations.current.delete(gameID);
        }
      }, { once: true });
      animation.addEventListener('cancel', () => {
        if (activeAnimations.current.get(gameID) === animation) {
          activeAnimations.current.delete(gameID);
        }
      }, { once: true });
    });

    previousPositions.current = nextPositions;
  }, [cancelActiveAnimations, measureCards]);

  const scheduleLayoutAnimation = useCallback(() => {
    if (animationFrame.current !== null) {
      window.cancelAnimationFrame(animationFrame.current);
    }

    animationFrame.current = window.requestAnimationFrame(() => {
      animationFrame.current = null;
      runLayoutAnimation();
    });
  }, [runLayoutAnimation]);

  const setCardRef = useCallback((gameID: number, cardElement: HTMLDivElement | null) => {
    if (cardElement === null) {
      cardRefs.current.delete(gameID);
      return;
    }

    cardRefs.current.set(gameID, cardElement);
  }, []);

  useLayoutEffect(() => {
    cancelActiveAnimations();
    previousPositions.current = measureCards();
  }, [cancelActiveAnimations, games, measureCards]);

  useLayoutEffect(() => {
    const gridElement = gridRef.current;

    if (gridElement === null) {
      return undefined;
    }

    const resizeObserver = new ResizeObserver(() => {
      scheduleLayoutAnimation();
    });

    resizeObserver.observe(gridElement);

    return () => {
      resizeObserver.disconnect();

      if (animationFrame.current !== null) {
        window.cancelAnimationFrame(animationFrame.current);
      }

      cancelActiveAnimations();
    };
  }, [cancelActiveAnimations, scheduleLayoutAnimation]);

  return (
    <div className="game-grid" ref={gridRef}>
      {games.map((game) => (
        <div className="game-grid-item" key={game.ID} ref={(cardElement) => setCardRef(game.ID, cardElement)}>
          <GameCard game={game} />
        </div>
      ))}
    </div>
  );
};
