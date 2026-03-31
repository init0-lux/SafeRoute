import React, { useState, useEffect } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    ScrollView,
    TextInput,
    Alert,
    Dimensions
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import BottomNavBar from '@/components/BottomNavBar';
import { useAuth } from '@/context/AuthContext';
import { IncidentType, INCIDENT_TYPES_MAP } from '@/constants/config';
import IncidentTypeSelector from '@/components/IncidentTypeSelector';
import { createReport } from '@/services/reports';
import * as Location from 'expo-location';

const { width } = Dimensions.get('window');

export default function HomeScreen() {
    const { user, logout } = useAuth();
    const [activeTab, setActiveTab] = useState("home");

    // Report Form State
    const [selectedTypes, setSelectedTypes] = useState<IncidentType[]>([]);
    const [selectorVisible, setSelectorVisible] = useState(false);
    const [description, setDescription] = useState('');
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [location, setLocation] = useState<Location.LocationObject | null>(null);
    const [locationLoading, setLocationLoading] = useState(true);

    useEffect(() => {
        (async () => {
            const { status } = await Location.requestForegroundPermissionsAsync();
            if (status !== 'granted') {
                setLocationLoading(false);
                return;
            }
            const currLocation = await Location.getCurrentPositionAsync({});
            setLocation(currLocation);
            setLocationLoading(false);
        })();
    }, []);

    const onTabPress = (tab: string) => {
        if (tab === 'sos') {
            router.push('/sos' as any);
        } else if (tab === 'location') {
            router.push('/maps' as any);
        } else if (tab === 'history') {
            router.push('/history' as any);
        } else {
            setActiveTab(tab);
        }
    };

    const handleToggleType = (type: IncidentType) => {
        setSelectedTypes((prev) =>
            prev.includes(type)
                ? prev.filter((t) => t !== type)
                : [...prev, type]
        );
    };

    const handleSubmitReport = async () => {
        if (selectedTypes.length === 0) {
            Alert.alert('Required', 'Please select an incident type');
            return;
        }

        if (!location) {
            Alert.alert('Location Required', 'Please wait for your location to be determined.');
            return;
        }

        const lat = location.coords.latitude;
        const lng = location.coords.longitude;

        const apiType = INCIDENT_TYPES_MAP[selectedTypes[0]] || selectedTypes[0];

        setIsSubmitting(true);
        try {
            await createReport({
                type: apiType,
                description: description.trim(),
                lat,
                lng,
            });
            Alert.alert('Report Submitted', 'Your report has been securely logged.');
            setSelectedTypes([]);
            setDescription('');
        } catch (err) {
            Alert.alert('Submission Failed', 'Failed to submit report. Please try again.');
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <View style={styles.container}>
            <ScrollView
                style={styles.scrollView}
                contentContainerStyle={styles.scrollContent}
                showsVerticalScrollIndicator={false}
            >
                {/* Top Header */}
                <View style={styles.header}>
                    <View style={styles.headerLeft}>
                        <View style={styles.avatar}>
                            <Ionicons name="person" size={24} color={Colors.bgDark} />
                        </View>
                        <View style={styles.greeting}>
                            <Text style={styles.greetingName}>
                                hello, {user?.username ?? 'user'}
                            </Text>
                            <Text style={styles.greetingSub}>let us help you find the safest route</Text>
                        </View>
                    </View>
                    <View style={styles.headerRight}>
                        <TouchableOpacity
                            onPress={() => router.push('/contacts' as any)}
                            style={styles.headerIcon}
                        >
                            <Ionicons name="people-outline" size={24} color={Colors.white} />
                        </TouchableOpacity>
                        <TouchableOpacity
                            onPress={async () => {
                                await logout();
                                router.replace('/login' as never);
                            }}
                            style={styles.headerIcon}
                        >
                            <Ionicons name="log-out-outline" size={24} color={Colors.white} />
                        </TouchableOpacity>
                    </View>
                </View>

                {/* Search Bar */}
                <View style={styles.searchContainer}>
                    <View style={styles.searchPill}>
                        <Ionicons name="search" size={20} color={Colors.gray} style={{ marginRight: Spacing.sm }} />
                        <TextInput
                            style={styles.searchInput}
                            placeholder="where are you going?"
                            placeholderTextColor={Colors.gray}
                            onFocus={() => router.push('/maps' as any)}
                        />
                    </View>
                    <View style={styles.searchPillShadow} />
                </View>

                {/* Map Card */}
                <TouchableOpacity
                    style={styles.mapCardContainer}
                    onPress={() => router.push('/maps' as any)}
                    activeOpacity={0.9}
                >
                    <View style={styles.mapCard}>
                        <View style={styles.mapPlaceholder}>
                            <Ionicons name="map" size={48} color={Colors.gray} />
                            <Text style={styles.mapPlaceholderText}>
                                {location ? 'tap to view safety map' : 'acquiring location...'}
                            </Text>
                        </View>
                        {/* Status Overlay */}
                        <View style={styles.mapStatus}>
                            <View style={[styles.statusDot, location && { backgroundColor: Colors.success }]} />
                            <Text style={styles.statusText}>
                                {location ? 'LIVE: Location Active' : 'Waiting for GPS...'}
                            </Text>
                        </View>
                    </View>
                </TouchableOpacity>

                {/* Neo-brutalist Report Section */}
                <View style={styles.reportSection}>
                    <Text style={styles.reportTitle}>Report The Incident Here</Text>

                    {/* Nature of incident */}
                    <TouchableOpacity
                        style={styles.brutalInput}
                        onPress={() => setSelectorVisible(true)}
                        activeOpacity={0.7}
                    >
                        <Text style={[styles.inputText, selectedTypes.length > 0 && styles.inputTextActive]}>
                            {selectedTypes.length > 0 ? selectedTypes.join(', ') : 'Nature of incident'}
                            {selectedTypes.length === 0 && <Text style={styles.required}>*</Text>}
                        </Text>
                        <Ionicons name="chevron-down" size={20} color={Colors.bgDark} />
                    </TouchableOpacity>

                    {/* Details */}
                    <TextInput
                        style={styles.brutalTextArea}
                        placeholder="provide details here"
                        placeholderTextColor={Colors.gray}
                        value={description}
                        onChangeText={setDescription}
                        multiline
                        numberOfLines={4}
                        textAlignVertical="top"
                    />

                    {/* Upload evidence */}
                    <TouchableOpacity style={styles.brutalInput} activeOpacity={0.7}>
                        <Text style={styles.inputText}>upload evidence</Text>
                        <Ionicons name="cloud-upload-outline" size={20} color={Colors.bgDark} />
                    </TouchableOpacity>

                    {/* Submit Button */}
                    <View style={styles.submitContainer}>
                        <TouchableOpacity
                            style={[
                                styles.brutalSubmit,
                                (selectedTypes.length === 0 || isSubmitting || !location) && styles.submitDisabled
                            ]}
                            onPress={handleSubmitReport}
                            disabled={isSubmitting || !location}
                            activeOpacity={0.85}
                        >
                            <Text style={styles.submitText}>
                                {isSubmitting ? 'submitting...' : 'submit'}
                            </Text>
                            <View style={styles.submitArrow}>
                                <Ionicons name="arrow-forward" size={16} color={Colors.white} />
                            </View>
                        </TouchableOpacity>
                    </View>
                </View>

            </ScrollView>
            <View style={styles.bottomNav}>
                <BottomNavBar activeTab={activeTab} onTabPress={onTabPress} />
            </View>

            {/* Incident Type Selector Modal */}
            <IncidentTypeSelector
                selectedTypes={selectedTypes}
                onToggleType={handleToggleType}
                visible={selectorVisible}
                onClose={() => setSelectorVisible(false)}
            />
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.bgDark,
    },
    scrollView: {
        flex: 1,
    },
    scrollContent: {
        paddingBottom: 140,
    },
    header: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: Spacing.lg,
        paddingTop: 60,
        paddingBottom: Spacing.lg,
    },
    headerLeft: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: Spacing.md,
    },
    headerRight: {
        flexDirection: 'row',
        gap: Spacing.sm,
    },
    headerIcon: {
        width: 40,
        height: 40,
        justifyContent: 'center',
        alignItems: 'center',
    },
    avatar: {
        width: 44,
        height: 44,
        borderRadius: 22,
        backgroundColor: Colors.white,
        borderWidth: 2,
        borderColor: Colors.bgDark,
        justifyContent: 'center',
        alignItems: 'center',
    },
    greeting: {
        justifyContent: 'center',
    },
    greetingName: {
        fontSize: FontSizes.md,
        fontWeight: '800',
        color: Colors.white,
        textTransform: 'lowercase',
    },
    greetingSub: {
        fontSize: 10,
        color: Colors.grayLight,
        marginTop: 2,
    },
    searchContainer: {
        marginHorizontal: Spacing.lg,
        marginBottom: Spacing.xl,
        position: 'relative',
    },
    searchPill: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        borderRadius: BorderRadius.full,
        paddingHorizontal: Spacing.lg,
        paddingVertical: 12,
        zIndex: 2,
    },
    searchPillShadow: {
        position: 'absolute',
        top: 6,
        left: 0,
        right: 0,
        bottom: -6,
        backgroundColor: Colors.bgDark,
        borderRadius: BorderRadius.full,
        zIndex: 1,
        opacity: 0.3,
    },
    searchInput: {
        flex: 1,
        fontSize: FontSizes.sm,
        fontWeight: '600',
        color: Colors.bgDark,
    },
    mapCardContainer: {
        marginHorizontal: Spacing.lg,
        marginBottom: Spacing.xl,
    },
    mapCard: {
        height: 180,
        backgroundColor: '#1A1A1A',
        borderRadius: 24,
        borderWidth: 3,
        borderColor: Colors.white,
        overflow: 'hidden',
        justifyContent: 'center',
        alignItems: 'center',
    },
    mapPlaceholder: {
        alignItems: 'center',
        gap: Spacing.sm,
    },
    mapPlaceholderText: {
        color: Colors.gray,
        fontSize: FontSizes.xs,
        fontWeight: '600',
    },
    mapStatus: {
        position: 'absolute',
        top: 12,
        right: 12,
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: 'rgba(0,0,0,0.6)',
        paddingHorizontal: 10,
        paddingVertical: 4,
        borderRadius: BorderRadius.full,
        gap: 6,
    },
    statusDot: {
        width: 8,
        height: 8,
        borderRadius: 4,
        backgroundColor: Colors.gray,
    },
    statusText: {
        color: Colors.white,
        fontSize: 10,
        fontWeight: '700',
    },
    reportSection: {
        marginHorizontal: Spacing.lg,
        backgroundColor: '#e2ccf5', // light purple from mockup
        borderRadius: 32,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        padding: Spacing.xl,
    },
    reportTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '900',
        color: Colors.bgDark,
        marginBottom: Spacing.lg,
        textAlign: 'center',
    },
    brutalInput: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        borderRadius: 16,
        paddingHorizontal: Spacing.lg,
        paddingVertical: 14,
        marginBottom: Spacing.md,
    },
    brutalTextArea: {
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        borderRadius: 16,
        paddingHorizontal: Spacing.lg,
        paddingVertical: 14,
        marginBottom: Spacing.md,
        minHeight: 100,
        fontSize: FontSizes.sm,
        color: Colors.bgDark,
        fontWeight: '600',
    },
    inputText: {
        fontSize: FontSizes.sm,
        color: Colors.gray,
        fontWeight: '700',
        flex: 1,
    },
    inputTextActive: {
        color: Colors.bgDark,
    },
    required: {
        color: Colors.sosRed,
    },
    submitContainer: {
        alignItems: 'center',
        marginTop: Spacing.sm,
    },
    brutalSubmit: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        borderRadius: BorderRadius.full,
        paddingLeft: Spacing.xl,
        paddingRight: 8,
        paddingVertical: 8,
        gap: Spacing.md,
        shadowColor: Colors.bgDark,
        shadowOffset: { width: 4, height: 4 },
        shadowOpacity: 1,
        shadowRadius: 0,
    },
    submitDisabled: {
        opacity: 0.6,
        shadowOffset: { width: 0, height: 0 },
    },
    submitText: {
        fontSize: FontSizes.md,
        fontWeight: '900',
        color: Colors.bgDark,
    },
    submitArrow: {
        width: 32,
        height: 32,
        borderRadius: 16,
        backgroundColor: Colors.bgDark,
        justifyContent: 'center',
        alignItems: 'center',
    },
    bottomNav: {
        position: "absolute",
        bottom: 20,
        left: 0,
        right: 0,
        flexDirection: "row",
        justifyContent: "center",
        alignItems: "center",
        zIndex: 100,
    },
});
