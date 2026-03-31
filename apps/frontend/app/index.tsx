import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Dimensions,
  FlatList,
  TouchableOpacity,
  Animated,
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import OnboardingSlide from '@/components/OnboardingSlide';
import PaginationDots from '@/components/PaginationDots';
import { Colors, Spacing } from '@/constants/theme';

const { width, height } = Dimensions.get('window');

// Badge/Checkmark SVG-like icon for slide 1
function PurpleBadgeIcon() {
  return (
    <View style={{ alignItems: 'center' }}>
      <View style={iconStyles.badgeLarge}>
        <Ionicons name="checkmark" size={64} color={Colors.bgDark} />
      </View>
      <View style={[iconStyles.badgeSmall, { position: 'absolute', left: -40, top: 100 }]}>
        <Ionicons name="checkmark" size={20} color={Colors.bgDark} />
      </View>
    </View>
  );
}

// Shield icon for slide 3
function OrangeShieldIcon() {
  return (
    <View style={{ alignItems: 'center' }}>
      <View style={iconStyles.shieldLarge}>
        <Ionicons name="shield-checkmark" size={80} color={Colors.bgDark} />
      </View>
      <View style={[iconStyles.shieldSmall, { position: 'absolute', left: -30, top: 90 }]}>
        <Ionicons name="shield-checkmark" size={24} color={Colors.bgDark} />
      </View>
    </View>
  );
}

const slides = [
  {
    id: '1',
    icon: <PurpleBadgeIcon />,
    title: 'Upload safe tamper-free evidence',
    highlightedWords: ['tamper-free'],
    subtitle: 'In accordance with Section 63(4)',
    accentColor: Colors.purple,
    dotColors: [Colors.purple, Colors.gray, Colors.grayDark],
  },
  {
    id: '2',
    icon: null,
    title: '3 beep SOS, connect with trusted contacts in under 30secs',
    highlightedWords: ['SOS,', 'connect', 'under', '30secs'],
    subtitle: 'we make sure you always get in touch with people you need',
    accentColor: Colors.red,
    dotColors: [Colors.purple, Colors.red, Colors.grayDark],
  },
  {
    id: '3',
    icon: <OrangeShieldIcon />,
    title: 'Find the safest routes in under 2mins',
    highlightedWords: ['safest', 'routes'],
    subtitle: 'because you deserve to enjoy the scenaries',
    accentColor: Colors.orange,
    dotColors: [Colors.purple, Colors.red, Colors.orange],
  },
];

export default function OnboardingScreen() {
  const [activeIndex, setActiveIndex] = useState(0);
  const [loopCount, setLoopCount] = useState(0);
  const flatListRef = useRef<FlatList>(null);
  const fadeAnim = useRef(new Animated.Value(1)).current;

  const goToNext = useCallback(() => {
    const nextIndex = (activeIndex + 1) % slides.length;

    if (nextIndex === 0) {
      const newLoopCount = loopCount + 1;
      setLoopCount(newLoopCount);
      if (newLoopCount >= 2) {
        router.replace('/home' as any);
        return;
      }
    }

    // Fade out then in for transition
    Animated.sequence([
      Animated.timing(fadeAnim, {
        toValue: 0,
        duration: 200,
        useNativeDriver: true,
      }),
      Animated.timing(fadeAnim, {
        toValue: 1,
        duration: 200,
        useNativeDriver: true,
      }),
    ]).start();

    setActiveIndex(nextIndex);
    flatListRef.current?.scrollToIndex({ index: nextIndex, animated: true });
  }, [activeIndex, loopCount, fadeAnim]);

  useEffect(() => {
    const timer = setInterval(goToNext, 3000);
    return () => clearInterval(timer);
  }, [goToNext]);

  const handleSkip = () => {
    router.replace('/home' as any);
  };

  return (
    <View style={styles.container}>
      <TouchableOpacity style={styles.skipButton} onPress={handleSkip} activeOpacity={0.7}>
        <Text style={styles.skipText}>Skip</Text>
        <Ionicons name="chevron-forward" size={16} color={Colors.grayLight} />
      </TouchableOpacity>

      <Animated.View style={[styles.slideContainer, { opacity: fadeAnim }]}>
        <FlatList
          ref={flatListRef}
          data={slides}
          horizontal
          pagingEnabled
          scrollEnabled={false}
          showsHorizontalScrollIndicator={false}
          keyExtractor={(item) => item.id}
          renderItem={({ item }) => (
            <OnboardingSlide
              icon={item.icon}
              title={item.title}
              highlightedWords={item.highlightedWords}
              subtitle={item.subtitle}
              accentColor={item.accentColor}
            />
          )}
          getItemLayout={(_, index) => ({
            length: width,
            offset: width * index,
            index,
          })}
        />
      </Animated.View>

      <View style={styles.paginationContainer}>
        <PaginationDots
          activeIndex={activeIndex}
          colors={slides[activeIndex].dotColors}
        />
      </View>
    </View>
  );
}

const iconStyles = StyleSheet.create({
  badgeLarge: {
    width: 140,
    height: 140,
    borderRadius: 70,
    backgroundColor: Colors.purple,
    justifyContent: 'center',
    alignItems: 'center',
    // Create wavy border effect
    borderWidth: 0,
    shadowColor: Colors.purple,
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0.4,
    shadowRadius: 20,
    elevation: 10,
  },
  badgeSmall: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: Colors.purple,
    justifyContent: 'center',
    alignItems: 'center',
    opacity: 0.7,
  },
  shieldLarge: {
    width: 140,
    height: 160,
    borderRadius: 20,
    backgroundColor: Colors.orange,
    justifyContent: 'center',
    alignItems: 'center',
    shadowColor: Colors.orange,
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0.4,
    shadowRadius: 20,
    elevation: 10,
  },
  shieldSmall: {
    width: 40,
    height: 44,
    borderRadius: 10,
    backgroundColor: Colors.orange,
    justifyContent: 'center',
    alignItems: 'center',
    opacity: 0.7,
  },
});

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.bgDark,
  },
  skipButton: {
    position: 'absolute',
    top: 60,
    right: Spacing.lg,
    zIndex: 10,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    paddingVertical: Spacing.sm,
    paddingHorizontal: Spacing.md,
  },
  skipText: {
    fontSize: 14,
    color: Colors.grayLight,
    fontWeight: '500',
  },
  slideContainer: {
    flex: 1,
  },
  paginationContainer: {
    position: 'absolute',
    bottom: 60,
    left: 0,
    right: 0,
    alignItems: 'center',
  },
});
